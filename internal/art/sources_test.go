package art

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stronxyo/quarter/internal/webapi"
)

// pngBytes is a small valid PNG, made in the test: no real logos in the repo.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, ellipse(40, 20)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalLogo(t *testing.T) {
	steam, dir := t.TempDir(), t.TempDir()
	cache := filepath.Join(steam, "appcache", "librarycache")
	logo := pngBytes(t)
	writeFile(t, filepath.Join(cache, "10", "logo.png"), logo)
	writeFile(t, filepath.Join(cache, "10", "abc123", "logo.png"), logo) // the direct file wins
	writeFile(t, filepath.Join(cache, "20", "abc123", "logo.png"), logo)
	writeFile(t, filepath.Join(dir, "30.png"), logo)
	s := NewSources(steam, dir, "", time.Second, 1<<20, time.Hour)

	for id, want := range map[uint32]string{
		10: filepath.Join(cache, "10", "logo.png"),
		20: filepath.Join(cache, "20", "abc123", "logo.png"),
		30: filepath.Join(dir, "30.png"),
	} {
		if got, ok := s.Local(id); !ok || got != want {
			t.Errorf("app %d: got %q, %v; want %q", id, got, ok, want)
		}
	}
	if _, ok := s.Local(40); ok {
		t.Error("app 40 has no logo on disk")
	}
}

// cdn serves a logo for app 1, a 404 for app 2, a big file for app 3, an
// HTML page for app 4 and a server error for app 5, counting requests.
func cdn(t *testing.T, logo []byte) (string, *atomic.Int32) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/1/logo.png":
			w.Write(logo)
		case "/3/logo.png":
			w.Write(append(logo, make([]byte, 2000)...))
		case "/4/logo.png":
			w.Write([]byte("<html>not a logo</html>"))
		case "/5/logo.png":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/%d/logo.png", &hits
}

func TestDownload(t *testing.T) {
	logo := pngBytes(t)
	url, _ := cdn(t, logo)
	dir := filepath.Join(t.TempDir(), "logos") // made on first download
	s := NewSources("", dir, url, time.Second, int64(len(logo)+100), time.Hour)

	p, err := s.Download(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, logo) {
		t.Error("the download should be saved as is")
	}
	if got, ok := s.Local(1); !ok || got != p {
		t.Errorf("a downloaded logo should be found locally next time, got %q", got)
	}
}

func TestDownloadMisses(t *testing.T) {
	logo := pngBytes(t)
	url, hits := cdn(t, logo)
	dir := t.TempDir()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	s := NewSources("", dir, url, time.Second, int64(len(logo)+100), 7*24*time.Hour)
	s.now = func() time.Time { return now }

	for _, id := range []uint32{2, 3, 4} { // 404, too big, not a PNG
		if _, err := s.Download(context.Background(), id); !errors.Is(err, ErrNoLogo) {
			t.Errorf("app %d: got %v, want ErrNoLogo", id, err)
		}
		if !s.Missing(id) {
			t.Errorf("app %d should be remembered as missing", id)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "3.png")); err == nil {
		t.Error("an oversized download must not be saved")
	}

	// Remembered across runs, until the retry time passes.
	next := NewSources("", dir, url, time.Second, 1<<20, 7*24*time.Hour)
	next.now = func() time.Time { return now.Add(6 * 24 * time.Hour) }
	if !next.Missing(2) {
		t.Error("the missing list should survive a restart")
	}
	next.now = func() time.Time { return now.Add(8 * 24 * time.Hour) }
	if next.Missing(2) {
		t.Error("after a week, the CDN should be asked again")
	}
	if hits.Load() != 3 {
		t.Errorf("want one request per app, got %d", hits.Load())
	}
}

// Network trouble is not remembered: the next selection tries again.
func TestDownloadNetworkErrorNotRemembered(t *testing.T) {
	url, _ := cdn(t, pngBytes(t))
	s := NewSources("", t.TempDir(), url, time.Second, 1<<20, time.Hour)
	if _, err := s.Download(context.Background(), 5); !errors.Is(err, webapi.ErrServer) {
		t.Errorf("a server error should say so, got %v", err)
	}
	if s.Missing(5) {
		t.Error("a server error must not be remembered as missing")
	}

	down := NewSources("", t.TempDir(), "http://127.0.0.1:1/%d/logo.png", time.Second, 1<<20, time.Hour)
	if _, err := down.Download(context.Background(), 1); !errors.Is(err, webapi.ErrUnreachable) || down.Missing(1) {
		t.Errorf("no network should fail as unreachable and not be remembered, got %v", err)
	}
}

// The CDN asking for fewer requests is a rate limit, not a missing logo.
func TestDownloadRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	s := NewSources("", t.TempDir(), srv.URL+"/%d/logo.png", time.Second, 1<<20, time.Hour)
	_, err := s.Download(context.Background(), 1)
	var limited *webapi.RateLimitError
	if !errors.As(err, &limited) || limited.RetryAfter != 30*time.Second || s.Missing(1) {
		t.Errorf("got %v, want a 30 s rate limit, not remembered", err)
	}
}

// When Steam's own logo turns out unusable, Quarter's download comes next.
func TestPathsOrder(t *testing.T) {
	steam, dir := t.TempDir(), t.TempDir()
	steamLogo := filepath.Join(steam, "appcache", "librarycache", "10", "logo.png")
	writeFile(t, steamLogo, pngBytes(t))
	writeFile(t, filepath.Join(dir, "10.png"), pngBytes(t))
	s := NewSources(steam, dir, "", time.Second, 1<<20, time.Hour)
	got := s.Paths(10)
	if len(got) != 2 || got[0] != steamLogo || got[1] != filepath.Join(dir, "10.png") {
		t.Errorf("got %v, want Steam's logo then Quarter's download", got)
	}
	if !s.Saved(10) || s.Saved(20) {
		t.Error("Saved should report Quarter's own downloads only")
	}
}

func TestRenderFile(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good.png"), filepath.Join(dir, "bad.png")
	writeFile(t, good, pngBytes(t))
	writeFile(t, bad, []byte("not a png"))
	if lines, err := RenderFile(good, 20, 5, Eighths); err != nil || len(lines) == 0 {
		t.Errorf("a good PNG should draw, got %v", err)
	}
	if _, err := RenderFile(bad, 20, 5, Eighths); err == nil {
		t.Error("a broken file should fail")
	}
	if _, err := RenderFile(filepath.Join(dir, "none.png"), 20, 5, Eighths); err == nil {
		t.Error("a missing file should fail")
	}
}

func TestNoCDN(t *testing.T) {
	s := NewSources("", t.TempDir(), "", time.Second, 1<<20, time.Hour)
	if _, err := s.Download(context.Background(), 1); !errors.Is(err, ErrNoLogo) {
		t.Errorf("without a CDN address, got %v", err)
	}
}

// Past the limit, a download removes the least recently used logos, never
// the new one; reading a saved logo marks it used.
func TestKeepAtMost(t *testing.T) {
	logo := pngBytes(t)
	url, _ := cdn(t, logo)
	dir := t.TempDir()
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, id := range []string{"10", "11", "12", "13"} { // 10 oldest
		p := filepath.Join(dir, id+".png")
		writeFile(t, p, logo)
		at := old.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(dir, missingFile), []byte("{}"))
	s := NewSources("", dir, url, time.Second, int64(len(logo)+100), time.Hour).KeepAtMost(3)

	s.Paths(10) // used now: no longer the oldest
	if _, err := s.Download(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	for id, kept := range map[string]bool{"1": true, "10": true, "11": false, "12": false, "13": true} {
		if _, err := os.Stat(filepath.Join(dir, id+".png")); (err == nil) != kept {
			t.Errorf("logo %s kept: %v, want %v", id, err == nil, kept)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, missingFile)); err != nil {
		t.Error("the list of games without a logo is not a logo: it stays")
	}
}
