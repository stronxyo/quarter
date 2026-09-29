// Package webapi talks to Steam over the network: the public store (no key)
// and, later, the Steam Web API (with the user's key).
package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrNotOnStore means the store has no page for the app (delisted, private,
// or not a store app at all).
var ErrNotOnStore = errors.New("not on the Steam store")

// StoreInfo is what the store says about one app.
type StoreInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "game", "demo", "dlc", ...
	Developers  []string `json:"developers"`
	Publishers  []string `json:"publishers"`
	Genres      []string `json:"genres"`
	ReleaseDate string   `json:"release_date"` // as the store writes it, e.g. "6 Jul, 2015"
	Description string   `json:"description"`  // plain text

	Achievements int `json:"achievements"` // how many the game has on Steam; 0 for none
}

// Year is the four-digit year in ReleaseDate, or "" if there isn't one.
func (s StoreInfo) Year() string {
	return yearPattern.FindString(s.ReleaseDate)
}

var yearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)

// StoreEntry is a cached store answer: the info, or a remembered failure.
type StoreEntry struct {
	Info    StoreInfo `json:"info"`
	Failed  bool      `json:"failed"`
	Fetched time.Time `json:"fetched"`
}

// Fresh reports whether the entry can be used as is: successes for
// refreshAfter, failures (so a delisted game isn't asked about again and
// again) for retryAfter.
func (e StoreEntry) Fresh(now time.Time, refreshAfter, retryAfter time.Duration) bool {
	if e.Failed {
		return now.Sub(e.Fetched) < retryAfter
	}
	return now.Sub(e.Fetched) < refreshAfter
}

// StoreClient fetches app details from the Steam store, never faster than
// one request per gap, however many callers share it.
type StoreClient struct {
	baseURL string
	http    *http.Client

	mu   sync.Mutex
	gap  time.Duration
	next time.Time
}

// NewStoreClient returns a client for baseURL (the appdetails endpoint).
func NewStoreClient(baseURL string, timeout, gap time.Duration) *StoreClient {
	return &StoreClient{baseURL: baseURL, http: &http.Client{Timeout: timeout}, gap: gap}
}

// wait blocks until this caller's turn under the rate limit, or ctx ends.
func (c *StoreClient) wait(ctx context.Context) error {
	c.mu.Lock()
	now := time.Now()
	at := c.next
	if at.Before(now) {
		at = now
	}
	c.next = at.Add(c.gap)
	c.mu.Unlock()

	select {
	case <-time.After(time.Until(at)):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AppDetails returns the store's details for appID.
func (c *StoreClient) AppDetails(ctx context.Context, appID uint32) (StoreInfo, error) {
	if err := c.wait(ctx); err != nil {
		return StoreInfo{}, err
	}
	q := url.Values{
		"appids":  {strconv.FormatUint(uint64(appID), 10)},
		"l":       {"english"},
		"filters": {"basic,developers,publishers,genres,release_date,achievements"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return StoreInfo{}, err
	}
	what := fmt.Sprintf("store %d", appID)
	resp, err := c.http.Do(req)
	if err != nil {
		return StoreInfo{}, Unreachable(what, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return StoreInfo{}, Refused(what, resp, time.Now(), http.StatusTooManyRequests, http.StatusForbidden)
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		// A cut-off download, or a page that isn't the store's (a hotel
		// network's sign-in page): says nothing about the game.
		return StoreInfo{}, fmt.Errorf("%s: %w: %w", what, ErrServer, err)
	}
	return parseAppDetails(body, appID)
}

// rawDetails is the part of an appdetails answer Quarter reads.
type rawDetails struct {
	Success bool `json:"success"`
	Data    struct {
		Type             string   `json:"type"`
		Name             string   `json:"name"`
		SteamAppID       uint32   `json:"steam_appid"`
		ShortDescription string   `json:"short_description"`
		Developers       []string `json:"developers"`
		Publishers       []string `json:"publishers"`
		Genres           []struct {
			Description string `json:"description"`
		} `json:"genres"`
		ReleaseDate struct {
			Date string `json:"date"`
		} `json:"release_date"`
		Achievements struct {
			Total int `json:"total"`
		} `json:"achievements"`
	} `json:"data"`
}

// parseAppDetails reads the single entry in an answer. The store sometimes
// files it under a different id than the one asked for (Rocket League 252950
// comes back as "1351431"), so the entry is matched on data.steam_appid.
func parseAppDetails(body map[string]json.RawMessage, appID uint32) (StoreInfo, error) {
	if len(body) != 1 {
		return StoreInfo{}, fmt.Errorf("store %d: %d entries in the answer, want 1", appID, len(body))
	}
	var raw rawDetails
	for _, v := range body {
		if err := json.Unmarshal(v, &raw); err != nil {
			return StoreInfo{}, fmt.Errorf("store %d: %w", appID, err)
		}
	}
	if !raw.Success {
		return StoreInfo{}, ErrNotOnStore
	}
	if raw.Data.SteamAppID != appID {
		return StoreInfo{}, fmt.Errorf("store %d: answer is for app %d", appID, raw.Data.SteamAppID)
	}

	info := StoreInfo{
		Name:        plainText(raw.Data.Name),
		Type:        raw.Data.Type,
		Developers:  raw.Data.Developers,
		Publishers:  raw.Data.Publishers,
		ReleaseDate: raw.Data.ReleaseDate.Date,
		Description: plainText(raw.Data.ShortDescription),

		Achievements: raw.Data.Achievements.Total,
	}
	for _, g := range raw.Data.Genres {
		info.Genres = append(info.Genres, g.Description)
	}
	return info, nil
}

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// plainText turns the store's HTML snippets into plain text.
func plainText(s string) string {
	s = tagPattern.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}
