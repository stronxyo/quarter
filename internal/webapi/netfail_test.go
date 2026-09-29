package webapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// answering serves every request with status, a Retry-After when given,
// and body.
func answering(t *testing.T, status int, retryAfter, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// unreachable is an address nothing answers on: a server, closed.
func unreachable(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
}

func TestStoreFailureKinds(t *testing.T) {
	var limited *RateLimitError
	tests := []struct {
		name  string
		url   string
		check func(error) bool
	}{
		{"no answer", unreachable(t), func(err error) bool { return errors.Is(err, ErrUnreachable) }},
		{"429 with seconds", answering(t, 429, "120", ""), func(err error) bool {
			return errors.As(err, &limited) && limited.RetryAfter == 2*time.Minute
		}},
		{"403 without Retry-After", answering(t, 403, "", ""), func(err error) bool {
			return errors.As(err, &limited) && limited.RetryAfter == 0 && limited.Status == 403
		}},
		{"500", answering(t, 500, "", ""), func(err error) bool { return errors.Is(err, ErrServer) }},
		{"a sign-in page instead of the store", answering(t, 200, "", "<html>Welcome to the hotel</html>"), func(err error) bool {
			return errors.Is(err, ErrServer)
		}},
		{"not on the store", answering(t, 200, "", `{"620":{"success":false}}`), func(err error) bool {
			return errors.Is(err, ErrNotOnStore)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStoreClient(tt.url, time.Second, 0).AppDetails(context.Background(), 620)
			if !tt.check(err) {
				t.Errorf("got %v", err)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"30", 30 * time.Second},
		{" 5 ", 5 * time.Second},
		{"-4", 0},
		{"Tue, 29 Sep 2026 12:02:00 GMT", 2 * time.Minute},
		{"Tue, 29 Sep 2026 11:00:00 GMT", 0}, // already past
		{"soon", 0},
	}
	for _, tt := range tests {
		if got := retryAfter(tt.in, now); got != tt.want {
			t.Errorf("retryAfter(%q): got %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestUnreachableKeepsCancel(t *testing.T) {
	err := Unreachable("store 1", context.Canceled)
	if errors.Is(err, ErrUnreachable) || !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled request is the caller's choice, not a failure: %v", err)
	}
}
