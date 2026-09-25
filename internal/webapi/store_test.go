package webapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStore serves the saved answers in testdata/store by appids.
func fakeStore(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.URL.RawQuery)
		mu.Unlock()
		name := map[string]string{
			"252950":  "appdetails_252950.json",
			"4551650": "appdetails_4551650.json",
			"1":       "appdetails_missing.json",
			"42":      "appdetails_mismatch.json",
		}[r.URL.Query().Get("appids")]
		if name == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := os.ReadFile("../../testdata/store/" + name)
		if err != nil {
			t.Error(err)
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

func TestAppDetails(t *testing.T) {
	srv, asked := fakeStore(t)
	c := NewStoreClient(srv.URL, 5*time.Second, 0)

	info, err := c.AppDetails(context.Background(), 252950)
	if err != nil {
		t.Fatalf("AppDetails: %v", err)
	}
	if info.Name != "Rocket League®" || info.Type != "game" || info.Year() != "2015" ||
		strings.Join(info.Developers, ",") != "Psyonix LLC" || strings.Join(info.Genres, ",") != "Action,Indie,Racing,Sports" {
		t.Errorf("parsed wrong: %+v", info)
	}
	if info.Description != "A made-up description for tests: cars play football & score goals." {
		t.Errorf("description should be plain text, got %q", info.Description)
	}
	q := (*asked)[0]
	for _, want := range []string{"appids=252950", "l=english", "filters=basic%2Cdevelopers%2Cpublishers%2Cgenres%2Crelease_date"} {
		if !strings.Contains(q, want) {
			t.Errorf("request %q missing %q", q, want)
		}
	}
}

func TestAppDetailsDemo(t *testing.T) {
	srv, _ := fakeStore(t)
	info, err := NewStoreClient(srv.URL, 5*time.Second, 0).AppDetails(context.Background(), 4551650)
	if err != nil || info.Type != "demo" || len(info.Genres) != 0 || info.Description != "" {
		t.Errorf("demo: got %+v, %v", info, err)
	}
}

func TestAppDetailsFailures(t *testing.T) {
	srv, _ := fakeStore(t)
	c := NewStoreClient(srv.URL, 5*time.Second, 0)

	if _, err := c.AppDetails(context.Background(), 1); !errors.Is(err, ErrNotOnStore) {
		t.Errorf("success:false should be ErrNotOnStore, got %v", err)
	}
	if _, err := c.AppDetails(context.Background(), 42); err == nil {
		t.Error("an answer for a different app must be rejected")
	}
	if _, err := c.AppDetails(context.Background(), 7); err == nil {
		t.Error("an HTTP error must be reported")
	}
}

func TestRateLimitSpacesRequests(t *testing.T) {
	srv, _ := fakeStore(t)
	c := NewStoreClient(srv.URL, 5*time.Second, 50*time.Millisecond)
	start := time.Now()
	for range 3 {
		if _, err := c.AppDetails(context.Background(), 252950); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Errorf("3 requests 50ms apart took %v, want at least 100ms", elapsed)
	}
}

func TestRateLimitRespectsCancel(t *testing.T) {
	c := NewStoreClient("http://example.invalid", time.Second, time.Hour)
	c.next = time.Now().Add(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.AppDetails(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

func TestEntryFresh(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	week, day := 7*24*time.Hour, 24*time.Hour
	tests := []struct {
		name string
		e    StoreEntry
		want bool
	}{
		{"new success", StoreEntry{Fetched: now.Add(-time.Hour)}, true},
		{"old success", StoreEntry{Fetched: now.Add(-8 * day)}, false},
		{"recent failure", StoreEntry{Failed: true, Fetched: now.Add(-time.Hour)}, true},
		{"old failure", StoreEntry{Failed: true, Fetched: now.Add(-2 * day)}, false},
	}
	for _, tt := range tests {
		if got := tt.e.Fresh(now, week, day); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestYear(t *testing.T) {
	for date, want := range map[string]string{"6 Jul, 2015": "2015", "Q4 2026": "2026", "Coming soon": "", "": ""} {
		if got := (StoreInfo{ReleaseDate: date}).Year(); got != want {
			t.Errorf("Year(%q): got %q, want %q", date, got, want)
		}
	}
}
