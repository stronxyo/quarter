package art

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/store"
	"github.com/stronxyo/quarter/internal/webapi"
)

// ErrNoLogo means Steam's CDN has no usable logo for the game: none at all,
// too big, or not a PNG. It is remembered, so the game isn't asked about
// again for a while.
var ErrNoLogo = errors.New("art: no logo on Steam's CDN")

// pngSignature starts every PNG file.
var pngSignature = []byte("\x89PNG\r\n\x1a\n")

const (
	missingFile    = "missing.json"
	missingVersion = 1
)

// Sources finds a game's logo: in Steam's own cache, in Quarter's earlier
// downloads, or on Steam's CDN. Safe for use from several goroutines.
type Sources struct {
	steamPath string
	dir       string // Quarter's logo folder
	cdn       string // logo address with %d for the app ID; "" for no downloads
	http      *http.Client
	maxBytes  int64
	retry     time.Duration // how long a missing logo is remembered
	now       func() time.Time

	keep int // downloaded logos kept at most, the least recently used going first; 0: no limit

	mu      sync.Mutex
	missing map[uint32]time.Time // loaded on first use
}

// KeepAtMost limits Quarter's logo folder to n downloaded logos: past that,
// each download removes the least recently used. A saved logo read for
// drawing counts as used (its file time is set to now). Steam's own cache
// is never touched.
func (s *Sources) KeepAtMost(n int) *Sources {
	s.keep = n
	return s
}

// NewSources returns Sources that read Steam's cache under steamPath and
// keep downloads, and the list of games without a logo, in dir.
func NewSources(steamPath, dir, cdn string, timeout time.Duration, maxBytes int64, retry time.Duration) *Sources {
	return &Sources{
		steamPath: steamPath,
		dir:       dir,
		cdn:       cdn,
		http:      &http.Client{Timeout: timeout},
		maxBytes:  maxBytes,
		retry:     retry,
		now:       time.Now,
	}
}

// Local returns the path of the first logo already on this PC.
func (s *Sources) Local(appID uint32) (string, bool) {
	if p := s.Paths(appID); len(p) > 0 {
		return p[0], true
	}
	return "", false
}

// Paths lists every logo already on this PC, best first: Steam's cache
// (newer clients keep it in a hashed subfolder), then Quarter's download.
// A caller whose first choice turns out unusable tries the next.
func (s *Sources) Paths(appID uint32) []string {
	var out []string
	if s.steamPath != "" {
		cache := steam.LibraryCacheDir(s.steamPath, appID)
		if p := filepath.Join(cache, "logo.png"); isFile(p) {
			out = append(out, p)
		} else if m, _ := filepath.Glob(filepath.Join(cache, "*", "logo.png")); len(m) > 0 {
			out = append(out, m[0])
		}
	}
	if p := s.saved(appID); isFile(p) {
		out = append(out, p)
		if s.keep > 0 {
			now := s.now()
			_ = os.Chtimes(p, now, now) // used: kept over older ones; failing only makes it go sooner
		}
	}
	return out
}

// Saved reports whether Quarter has downloaded the game's logo before.
func (s *Sources) Saved(appID uint32) bool { return isFile(s.saved(appID)) }

func (s *Sources) saved(appID uint32) string {
	return filepath.Join(s.dir, strconv.FormatUint(uint64(appID), 10)+".png")
}

// Missing reports whether the CDN recently had no logo for the game, so
// there's no point asking again yet.
func (s *Sources) Missing(appID uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.loadMissing()[appID]
	return ok && s.now().Sub(at) < s.retry
}

// Download fetches the game's logo from Steam's CDN, keeps it in Quarter's
// logo folder and returns its path. It returns ErrNoLogo (and remembers it)
// when the CDN has none. Network failures (classified as in webapi) are
// not remembered, so the next try goes to the network again.
func (s *Sources) Download(ctx context.Context, appID uint32) (string, error) {
	if s.cdn == "" {
		return "", ErrNoLogo
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(s.cdn, appID), nil)
	if err != nil {
		return "", err
	}
	what := fmt.Sprintf("logo %d", appID)
	resp, err := s.http.Do(req)
	if err != nil {
		return "", webapi.Unreachable(what, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden:
		return "", s.noLogo(appID)
	case resp.StatusCode != http.StatusOK:
		return "", webapi.Refused(what, resp, s.now(), http.StatusTooManyRequests)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, s.maxBytes+1))
	if err != nil {
		return "", webapi.Unreachable(what, err) // the connection dropped mid-download
	}
	if int64(len(b)) > s.maxBytes || !bytes.HasPrefix(b, pngSignature) {
		return "", s.noLogo(appID)
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	p := s.saved(appID)
	if err := store.WriteFile(s.dir, filepath.Base(p), b); err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	s.prune(filepath.Base(p))
	return p, nil
}

// prune removes the least recently used downloaded logos past the limit,
// never the one just saved.
func (s *Sources) prune(just string) {
	if s.keep <= 0 {
		return
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	type logo struct {
		name string
		used time.Time
	}
	var logos []logo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" || e.Name() == just {
			continue
		}
		if info, err := e.Info(); err == nil {
			logos = append(logos, logo{e.Name(), info.ModTime()})
		}
	}
	extra := len(logos) + 1 - s.keep
	if extra <= 0 {
		return
	}
	slices.SortFunc(logos, func(a, b logo) int { return a.used.Compare(b.used) })
	for _, l := range logos[:extra] {
		if err := os.Remove(filepath.Join(s.dir, l.name)); err != nil {
			slog.Warn("logo", "remove", l.name, "err", err)
		}
	}
}

// noLogo remembers that the CDN has no logo for appID and returns ErrNoLogo.
// Failing to save only means asking again next run.
func (s *Sources) noLogo(appID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	missing := s.loadMissing()
	missing[appID] = s.now()
	err := os.MkdirAll(s.dir, 0o755)
	if err == nil {
		err = store.Save(s.dir, missingFile, missingVersion, missing)
	}
	if err != nil {
		slog.Warn("logos", "err", err)
	}
	return ErrNoLogo
}

// loadMissing reads the missing list once. The caller holds s.mu.
func (s *Sources) loadMissing() map[uint32]time.Time {
	if s.missing == nil {
		m, ok := store.Load[map[uint32]time.Time](s.dir, missingFile, missingVersion)
		if !ok || m == nil {
			m = map[uint32]time.Time{}
		}
		s.missing = m
	}
	return s.missing
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
