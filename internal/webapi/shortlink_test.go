package webapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// shortener is a fake link-shortening service that redirects every link to
// to, counting what it is asked. Its host counts as a short-link service
// for the test.
func shortener(t *testing.T, to string, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		if r.Method != http.MethodHead {
			t.Errorf("a short link should be asked with HEAD, got %s", r.Method)
		}
		if to == "" {
			w.WriteHeader(status)
			return
		}
		http.Redirect(w, r, to, status)
	}))
	t.Cleanup(srv.Close)
	host, _ := url.Parse(srv.URL)
	shortHosts[host.Hostname()] = true
	t.Cleanup(func() { delete(shortHosts, host.Hostname()) })
	return srv, &asked
}

// A short link as the store's website becomes the official site it leads
// to, which is never visited itself.
func TestShortLinkFollowed(t *testing.T) {
	short, asked := shortener(t, "https://www.oncehuman.game/", http.StatusMovedPermanently)
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"2139460":{"success":true,"data":{"type":"game","name":"Once Human","steam_appid":2139460,"website":%q}}}`, short.URL+"/gRX2BR4tj9")
	}))
	t.Cleanup(store.Close)
	info, err := NewStoreClient(store.URL, 5*time.Second, 0).AppDetails(context.Background(), 2139460)
	if err != nil {
		t.Fatal(err)
	}
	if info.Website != "https://www.oncehuman.game/" || asked.Load() != 1 {
		t.Errorf("website %q after %d asks, want the official site after 1", info.Website, asked.Load())
	}
}

// Short links in a row are followed to the first address that isn't one;
// one that leads nowhere, loops, or can't be reached leaves no website.
func TestShortLinkEnds(t *testing.T) {
	c := NewStoreClient("", 5*time.Second, 0)
	ctx := context.Background()

	last, _ := shortener(t, "https://example.com/game", http.StatusFound)
	first, _ := shortener(t, last.URL+"/x", http.StatusMovedPermanently)
	if got := c.unshorten(ctx, first.URL+"/y"); got != "https://example.com/game" {
		t.Errorf("two short links in a row: got %q", got)
	}

	dead, _ := shortener(t, "", http.StatusNotFound)
	if got := c.unshorten(ctx, dead.URL+"/gone"); got != "" {
		t.Errorf("a dead short link: got %q, want none", got)
	}

	var loop *httptest.Server
	loop = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, loop.URL+"/again", http.StatusFound)
	}))
	t.Cleanup(loop.Close)
	host, _ := url.Parse(loop.URL)
	shortHosts[host.Hostname()] = true
	t.Cleanup(func() { delete(shortHosts, host.Hostname()) })
	if got := c.unshorten(ctx, loop.URL+"/a"); got != "" {
		t.Errorf("a loop of short links: got %q, want none", got)
	}

	gone, _ := shortener(t, "https://example.com/", http.StatusFound)
	gone.Close()
	if got := c.unshorten(ctx, gone.URL+"/z"); got != "" {
		t.Errorf("an unreachable short link: got %q, want none", got)
	}

	odd, _ := shortener(t, "file:///C:/Windows/system32", http.StatusFound)
	if got := c.unshorten(ctx, odd.URL+"/f"); got != "" {
		t.Errorf("a short link to something other than a web page: got %q, want none", got)
	}
}

func TestIsShortLink(t *testing.T) {
	for link, want := range map[string]bool{
		"https://t.co/gRX2BR4tj9":     true,
		"http://BIT.LY/abc":           true,
		"https://www.oncehuman.game/": false,
		"https://t.com/":              false,
		"":                            false,
	} {
		if got := isShortLink(link); got != want {
			t.Errorf("isShortLink(%q) = %v, want %v", link, got, want)
		}
	}
}
