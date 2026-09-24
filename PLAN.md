# Quarter: v0.1 Implementation Plan

Companion to `PROJECT_BRIEF.md`. Structure first, then milestones. Every milestone ends with something runnable on Windows.

## Project structure

```
quarter/
├─ cmd/quarter/main.go        # flags, wiring, starts the TUI (or --list plain output)
├─ internal/
│  ├─ vdf/                    # text VDF/ACF parser (+ tests)
│  ├─ steam/
│  │  ├─ locate_windows.go    # SteamPath from registry (golang.org/x/sys/windows/registry)
│  │  ├─ locate_other.go      # ~/.steam/steam, ~/Library/Application Support/Steam
│  │  ├─ libraries.go         # libraryfolders.vdf → library paths
│  │  ├─ manifests.go         # appmanifest_*.acf → installed apps
│  │  ├─ users.go             # loginusers.vdf → SteamID64 / accountid
│  │  ├─ localconfig.go       # localconfig.vdf → playtime, last played
│  │  ├─ filter.go            # non-game appid denylist
│  │  └─ launch.go            # steam:// URIs, opened via the OS handler
│  ├─ webapi/
│  │  ├─ client.go            # GetOwnedGames, GetPlayerAchievements
│  │  └─ store.go             # appdetails + rate limiter
│  ├─ library/
│  │  ├─ game.go              # Game model (merged view of all sources)
│  │  └─ merge.go             # local + API + store + favourites → []Game
│  ├─ cache/                  # JSON cache files + image cache, with timestamps
│  ├─ config/                 # paths (%APPDATA%\quarter), config.json, favourites
│  ├─ art/
│  │  ├─ source.go            # librarycache → disk cache → CDN
│  │  └─ halfblock.go         # image → "▀" truecolor string
│  └─ ui/
│     ├─ app.go               # root Bubbletea model, messages, layout
│     ├─ list.go              # library pane: search, sort, filter
│     ├─ detail.go            # detail pane
│     ├─ statusbar.go
│     ├─ help.go              # ? overlay
│     ├─ wizard.go            # first-run / API key setup
│     ├─ keys.go              # keymap (bubbles/key)
│     └─ styles.go            # Lip Gloss styles
└─ testdata/steam/            # fake Steam install: libraryfolders, manifests, loginusers, localconfig
```

Key decisions:
- **`internal/steam` knows nothing about the UI**, and `ui` never reads files directly. Data comes in as Bubbletea messages from background commands, so the UI never blocks.
- **`--steam-path` flag / `QUARTER_STEAM_PATH` env** overrides detection. This is what makes the scanner testable against `testdata/steam/` and debuggable on odd installs.
- **`quarter --list`** prints the scanned library as plain text. Useful for M1 and for bug reports forever after.
- **Launching** on Windows uses `rundll32 url.dll,FileProtocolHandler <uri>` (no console window flash, no `cmd` quoting issues). `open` on macOS, `xdg-open` on Linux.
- **Every local-format parser fails soft**: a bad `localconfig.vdf` means no playtime, not a crash.

### Game model (first cut)

```go
type Game struct {
    AppID        uint32
    Name         string
    Installed    bool
    LibraryPath  string
    SizeOnDisk   int64
    Playtime     time.Duration  // localconfig, overridden by API when present
    LastPlayed   time.Time
    Favourite    bool
    Achievements *Progress      // nil until fetched
    Meta         *StoreMeta     // nil until fetched
}
```

## Milestones

**M0: Setup**
- Install Go (not currently on this machine): `winget install GoLang.Go`
- `git init`, `go mod init <module path>`, add Bubbletea, Bubbles, Lip Gloss, sahilm/fuzzy
- ✅ `go run ./cmd/quarter` prints a version line

**M1: Scan**
- VDF parser, locate Steam, read libraries + manifests, denylist filter
- `loginusers.vdf` + `localconfig.vdf` for playtime and last played
- Fixture Steam dir in `testdata/` + unit tests for each parser
- ✅ `quarter --list` prints your real installed games with playtime

**M2: List UI**
- Library pane with fuzzy search (`/`), sort (`o`), filter (`Tab`), `j/k`, status bar, `q`
- Library index cached to JSON; launch shows cache instantly, rescans in background
- Startup benchmark (cache → interactive list) added to the repo, target < 100 ms
- ✅ Browse and search your real library in Windows Terminal

**M3: Actions**
- `Enter` play, `i` install, `s` store page, `f` favourite (persisted in config), `r` refresh
- ✅ Launch a game from Quarter

**Spike: Sixel (time-boxed, ~half a day)**
- Throwaway prototype: draw one sixel image beside a Bubbletea list, move selection, resize the window
- Decide: sixel as an opt-in mode in v0.1, or half-block only. Write the outcome into the brief.

**M4: Detail pane**
- Two-pane layout, responsive collapse to overlay on narrow terminals
- Store `appdetails` fetch for the selected game, rate limiter, cache (refresh window: days)
- ✅ Select a game, see developer / year / genres / description / playtime

**M5: Web API**
- Setup wizard (key only; SteamID64 auto-detected), plain-text storage notice
- `GetOwnedGames` merges in uninstalled games; private-profile detection and message
- Achievements fetched lazily for the selected game, progress bar
- ✅ Uninstalled games appear with `○`; achievements show in the detail pane

**M6: Cover art**
- Art source chain: `librarycache` → disk cache → CDN
- Half-block renderer (plus sixel, if the spike said yes)
- ✅ Cover art in the detail pane

**M7: Polish**
- `?` help overlay, "INSERT COIN" first-run splash, "GAME OVER" on quit
- Error states: Steam not found, no libraries, offline
- Check the performance targets from the brief (startup, 5,000-game search); `pprof` any misses
- GoReleaser config: Windows/macOS/Linux × amd64/arm64, stripped binaries
- Smoke test `quarter.exe` on a clean terminal; Linux build on a Steam Deck if one is available
- ✅ v0.1 tagged

## Testing

- Unit tests for `vdf`, `steam/*` parsers, `filter`, `library/merge`, run against `testdata/`
- `webapi` tested with `httptest` servers, never the real API in tests
- UI: key flows checked with `teatest` (search narrows list, filter cycles, Enter emits the launch URI) where cheap; the rest by hand in Windows Terminal

## Open questions

1. Go module path: `github.com/<your-user>/quarter`, or something else?
2. Should the first-run wizard appear automatically, or only from a key (e.g. `ctrl+k`) so the app opens straight to the list?
3. License (MIT vs GPL core + private Pro), decided before the repo goes public. See "License and monetization" in the brief.
