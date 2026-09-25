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
	"strconv"
	"sync"
	"time"

	"github.com/stronxyo/quarter/internal/store"
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

	mu      sync.Mutex
	missing map[uint32]time.Time // loaded on first use
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

// Local returns the path of a logo already on this PC: Steam's cache first
// (newer clients keep it in a hashed subfolder), then Quarter's download.
func (s *Sources) Local(appID uint32) (string, bool) {
	id := strconv.FormatUint(uint64(appID), 10)
	if s.steamPath != "" {
		cache := filepath.Join(s.steamPath, "appcache", "librarycache", id)
		if p := filepath.Join(cache, "logo.png"); isFile(p) {
			return p, true
		}
		if m, _ := filepath.Glob(filepath.Join(cache, "*", "logo.png")); len(m) > 0 {
			return m[0], true
		}
	}
	if p := filepath.Join(s.dir, id+".png"); isFile(p) {
		return p, true
	}
	return "", false
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
// when the CDN has none; network errors are returned as they are and not
// remembered, so the next try goes to the network again.
func (s *Sources) Download(ctx context.Context, appID uint32) (string, error) {
	if s.cdn == "" {
		return "", ErrNoLogo
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(s.cdn, appID), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden:
		return "", s.noLogo(appID)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("logo %d: HTTP %d", appID, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, s.maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	if int64(len(b)) > s.maxBytes || !bytes.HasPrefix(b, pngSignature) {
		return "", s.noLogo(appID)
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	name := strconv.FormatUint(uint64(appID), 10) + ".png"
	if err := store.WriteFile(s.dir, name, b); err != nil {
		return "", fmt.Errorf("logo %d: %w", appID, err)
	}
	return filepath.Join(s.dir, name), nil
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
