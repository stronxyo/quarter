package replay

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Answers in the shape Steam gives (trimmed): a year with per-game months,
// 2023's shape (months without games), and a private year.
const (
	fullYear = `{"response":{"stats":{"account_id":2,"year":2024,"privacy_state":3,"playtime_stats":{
		"total_stats":{"total_playtime_seconds":36000,"total_sessions":12},
		"games":[{"appid":252950,"stats":{"total_playtime_seconds":30000}},{"appid":620,"stats":{"total_playtime_seconds":6000}}],
		"months":[
			{"rtime_month":1704067200,"stats":{"total_playtime_seconds":7200,"total_sessions":3},
			 "appid":[{"appid":252950,"stats":{"total_playtime_seconds":3600}},{"appid":620,"stats":{"total_playtime_seconds":3600}}]},
			{"rtime_month":1706745600,"stats":{"total_playtime_seconds":0,"total_sessions":0}}
		]}},"performance_stats":{},"distribution":{}}}`
	monthsOnly  = `{"response":{"stats":{"playtime_stats":{"total_stats":{"total_playtime_seconds":7200},"months":[{"rtime_month":1672531200,"stats":{"total_playtime_seconds":7200,"total_sessions":2}}]}}}}`
	privateYear = `{"response":{}}`
)

func serve(t *testing.T, body string, status int) (*Client, *string) {
	t.Helper()
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RawQuery
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, time.Second), &asked
}

func TestFetchYear(t *testing.T) {
	c, asked := serve(t, fullYear, http.StatusOK)
	y, err := c.Fetch(context.Background(), 76561197960265730, 2024)
	if err != nil {
		t.Fatal(err)
	}
	if *asked != "steamid=76561197960265730&year=2024" {
		t.Errorf("only the SteamID and the year go to Steam, asked %q", *asked)
	}
	if y.Year != 2024 || y.Played != 10*time.Hour || y.Sessions != 12 || y.Games[252950] != 30000*time.Second {
		t.Errorf("year: %+v", y)
	}
	if len(y.Months) != 2 {
		t.Fatalf("want 2 months, got %d", len(y.Months))
	}
	jan := y.Months[0]
	if !jan.Start.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) || jan.Played != 2*time.Hour || jan.Sessions != 3 {
		t.Errorf("January: %+v", jan)
	}
	if jan.Games[252950] != time.Hour || jan.Games[620] != time.Hour {
		t.Errorf("January's games: %v", jan.Games)
	}
	if y.Months[1].Games != nil {
		t.Errorf("a month with no games has none: %v", y.Months[1].Games)
	}
}

// 2023's shape: monthly totals, no per-game months.
func TestFetchMonthsWithoutGames(t *testing.T) {
	c, _ := serve(t, monthsOnly, http.StatusOK)
	y, err := c.Fetch(context.Background(), 1, 2023)
	if err != nil || len(y.Months) != 1 || y.Months[0].Played != 2*time.Hour || len(y.Months[0].Games) != 0 {
		t.Errorf("got %+v, %v", y, err)
	}
}

func TestFetchNotPublic(t *testing.T) {
	c, _ := serve(t, privateYear, http.StatusOK)
	if _, err := c.Fetch(context.Background(), 1, 2025); !errors.Is(err, ErrNotPublic) {
		t.Errorf("a private year: got %v, want ErrNotPublic", err)
	}
}

func TestFetchFailures(t *testing.T) {
	c, _ := serve(t, "", http.StatusInternalServerError)
	if _, err := c.Fetch(context.Background(), 1, 2025); err == nil || errors.Is(err, ErrNotPublic) {
		t.Errorf("a server error is an error, not 'not public': %v", err)
	}
	c, _ = serve(t, "<html>sign in</html>", http.StatusOK)
	if _, err := c.Fetch(context.Background(), 1, 2025); err == nil || errors.Is(err, ErrNotPublic) {
		t.Errorf("a page that isn't Steam's answer is an error: %v", err)
	}
}
