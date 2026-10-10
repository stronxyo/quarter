package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/replay"
	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/store"
	"github.com/stronxyo/quarter/internal/ui"
	"github.com/stronxyo/quarter/internal/webapi"
)

// version is overridden at release time with -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	list := flag.Bool("list", false, "print installed games as plain text and exit")
	steamPath := flag.String("steam-path", "", "Steam install folder (default: detect)")
	flag.Parse()

	var err error
	switch {
	case *showVersion:
		fmt.Printf("%s %s\n", config.AppName, version)
	case *list:
		err = listGames(os.Stdout, os.Stderr, *steamPath)
	default:
		err = runUI(*steamPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", config.AppName, err)
		os.Exit(1)
	}
}

func runUI(steamPath string) error {
	dir, err := config.DataDir()
	if err != nil {
		// Without a data folder Quarter still runs, just without cache or log.
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		final, err := tea.NewProgram(ui.New(ui.Options{SteamPath: steamPath, Open: steam.OpenURI, BasicGlyphs: basicGlyphs("")})).Run()
		if err == nil {
			sayGoodbye(final, nil)
		}
		return err
	}

	closeLog := setupLog(dir)
	defer closeLog()

	save := func(r library.Result) error {
		return store.Save(dir, config.LibraryCache, config.LibraryCacheVersion, r)
	}
	settings := config.LoadSettings(dir)
	// The four saved files are read at the same time: on a large library,
	// reading them is most of the start (TestSpeedTargets' startup does the
	// same).
	var (
		cached     *library.Result
		sessions   []steam.Session
		years      []replay.Year
		storeCache map[uint32]webapi.StoreEntry
		wg         sync.WaitGroup
	)
	wg.Go(func() { cached = loadCachedLibrary(dir, steamPath) })
	wg.Go(func() { sessions = loadSessions(dir) })
	wg.Go(func() { years = loadReplay(dir) })
	wg.Go(func() { storeCache = loadStoreCache(dir) })
	wg.Wait()
	final, err := tea.NewProgram(ui.New(ui.Options{
		SteamPath: steamPath,
		Cached:    cached,
		Save:      save,
		Open:      steam.OpenURI,

		RecordSessions: sessionRecorder(dir),
		Sessions:       sessions,

		Replay:       years,
		FetchReplay:  replayFetcher(steamPath),
		SaveReplay:   func(y []replay.Year) error { return store.Save(dir, config.ReplayFile, config.ReplayVersion, y) },
		DeleteReplay: func() error { return removeIfThere(filepath.Join(dir, config.ReplayFile)) },
		RunningApp:   steam.RunningApp,

		// Steam starts in the tray when Quarter opens, and games start
		// without its window (specs/BATCH_08_SPEC.md, "Steam starts quietly").
		SteamClient: steam.ClientState,
		StartSteam:  func() error { return steam.StartQuiet(steamPath) },
		PlayQuiet:   func(id uint32) error { return steam.PlayQuiet(steamPath, id) },

		Settings:     settings,
		SaveSettings: func(s config.Settings) error { return config.SaveSettings(dir, s) },
		// The start screen on every start, led by a random intro (Pablo's
		// choice); config.json's "intro" turns the intros off.
		Splash: true,

		BasicGlyphs: basicGlyphs(settings.Glyphs),

		FetchStore: webapi.NewStoreClient(config.StoreURL, config.StoreTimeout, config.StoreGap).AppDetails,
		StoreCache: storeCache,
		SaveStore: func(c map[uint32]webapi.StoreEntry) error {
			return store.Save(dir, config.StoreCache, config.StoreCacheVersion, c)
		},

		Logos: func(steamPath string) ui.LogoSource {
			return art.NewSources(steamPath, filepath.Join(dir, config.LogoDir),
				config.LogoURL, config.StoreTimeout, config.LogoMaxBytes, config.LogoRetry).KeepAtMost(config.LogoKeep)
		},
	})).Run()
	if err != nil {
		return err
	}
	sayGoodbye(final, func(s config.Settings) error { return config.SaveSettings(dir, s) })
	return nil
}

// sayGoodbye prints the goodbye as Quarter closes, once the terminal is
// back, and saves which one it was so the next is another.
func sayGoodbye(final tea.Model, save func(config.Settings) error) {
	m, ok := final.(ui.Model)
	if !ok {
		return
	}
	text, s := m.Farewell(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 2)))
	if text == "" {
		return
	}
	if save != nil {
		if err := save(s); err != nil {
			slog.Warn("settings", "err", err)
		}
	}
	printGoodbye(text, colourOutput)
}

// printGoodbye prints the goodbye in colour where the output reads colour
// codes, and as plain text where it can't, never as the codes themselves
// (the classic console showed "←[38;2;..." before, 2026-10-02).
func printGoodbye(text string, colour func() (func(), bool)) {
	restore, ok := colour()
	defer restore()
	if !ok {
		text = ansi.Strip(text)
	}
	fmt.Print(text)
}

// sessionRecorder adds the game sessions in Steam's log to sessions.json in
// dir, and returns every session recorded, for the history screen. One
// record runs at a time, so two scans never race on the file.
func sessionRecorder(dir string) func(steamPath string) ([]steam.Session, error) {
	var mu sync.Mutex
	return func(steamPath string) ([]steam.Session, error) {
		found, err := steam.ReadSessions(steamPath, time.Local)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		saved, ok := store.Load[[]steam.Session](dir, config.SessionsFile, config.SessionsVersion)
		if !ok {
			// Unreadable history is kept aside, never overwritten.
			if _, err := store.SetAside(dir, config.SessionsFile); err != nil {
				return nil, err
			}
		}
		all, added := library.MergeSessions(saved, found)
		if added == 0 {
			return all, nil
		}
		return all, store.Save(dir, config.SessionsFile, config.SessionsVersion, all)
	}
}

// loadReplay returns the past years brought in on an earlier run; a
// missing or unreadable file means none, so the user can ask again.
func loadReplay(dir string) []replay.Year {
	y, _ := store.Load[[]replay.Year](dir, config.ReplayFile, config.ReplayVersion)
	return y
}

// replayFetcher asks Steam for each past year, from the first Steam Replay
// to last year, for the account signed in on this PC (only its public
// SteamID is sent). Years Steam gives nothing for (not public) are listed
// apart; a network failure stops it.
func replayFetcher(steamPath string) func(context.Context) ([]replay.Year, []int, error) {
	return func(ctx context.Context) ([]replay.Year, []int, error) {
		path, err := steam.Locate(steamPath)
		if err != nil {
			return nil, nil, err
		}
		acct, err := steam.ActiveAccount(path)
		if err != nil {
			return nil, nil, err
		}
		client := replay.NewClient(config.ReplayURL, config.ReplayTimeout)
		var years []replay.Year
		var missing []int
		for y := replay.FirstYear; y < time.Now().Year(); y++ {
			year, err := client.Fetch(ctx, acct.SteamID64, y)
			switch {
			case errors.Is(err, replay.ErrNotPublic):
				missing = append(missing, y)
			case err != nil:
				return nil, nil, err
			default:
				years = append(years, year)
			}
		}
		return years, missing, nil
	}
}

// removeIfThere deletes path; a file already gone is fine.
func removeIfThere(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// loadSessions returns the sessions recorded by earlier runs, so history
// shows at once; a missing or unreadable file means none yet.
func loadSessions(dir string) []steam.Session {
	s, _ := store.Load[[]steam.Session](dir, config.SessionsFile, config.SessionsVersion)
	return s
}

// loadStoreCache returns the store details saved by earlier runs; a missing or
// unreadable cache just means they are fetched again.
func loadStoreCache(dir string) map[uint32]webapi.StoreEntry {
	c, _ := store.Load[map[uint32]webapi.StoreEntry](dir, config.StoreCache, config.StoreCacheVersion)
	return c
}

// loadCachedLibrary returns the library saved by the last run, or nil. A cache
// from a different Steam folder than --steam-path asks for is ignored.
func loadCachedLibrary(dir, steamPath string) *library.Result {
	res, ok := store.Load[library.Result](dir, config.LibraryCache, config.LibraryCacheVersion)
	if !ok {
		return nil
	}
	if steamPath != "" && !strings.EqualFold(filepath.Clean(steamPath), res.SteamPath) {
		return nil
	}
	return &res
}

// setupLog sends slog output to quarter.log in the data folder, since the UI
// owns the terminal. If the file can't be created, logs are dropped.
func setupLog(dir string) (closeLog func()) {
	f, err := os.Create(filepath.Join(dir, config.LogFile))
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return func() {}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	return func() { f.Close() }
}

// listGames prints the library to out and any warnings to warn, so the list
// stays clean when redirected.
func listGames(out, warn io.Writer, steamPath string) error {
	res, err := library.Scan(steamPath)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(warn, "warning: %v\n", w)
	}

	summary := fmt.Sprintf("%d games", len(res.Games))
	width := utf8.RuneCountInString(summary)
	for _, g := range res.Games {
		width = max(width, utf8.RuneCountInString(g.Label()))
	}

	var total time.Duration
	for _, g := range res.Games {
		total += g.Playtime
		fmt.Fprintf(out, "%-*s  %9s  %10s\n", width, g.Label(), library.FormatHours(g.Playtime, res.PlaytimeKnown), library.FormatDate(g.LastPlayed))
	}
	fmt.Fprintf(out, "\n%-*s  %9s\n", width, summary, library.FormatHours(total, res.PlaytimeKnown))
	return nil
}
