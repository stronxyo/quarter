// Package replay reads a Steam user's Steam Replay (Valve's year in review,
// 2022 on): playtime per month, per game where Steam gives it. It uses
// Steam's public Web API with only the user's SteamID: no login, no key.
// The method isn't in Valve's documentation (it's defined in Steam's own
// protobufs, service_salefeature.proto), so callers treat it as optional.
package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// FirstYear is the first year Steam made a Replay for.
const FirstYear = 2022

// ErrNotPublic means Steam gave no Replay for the year: the user hasn't set
// it to Public (the default is private), or there is none.
var ErrNotPublic = errors.New("replay not public")

// Month is one month of play.
type Month struct {
	Start    time.Time                `json:"start"` // the month's first day, UTC
	Played   time.Duration            `json:"played"`
	Sessions int                      `json:"sessions"`
	Games    map[uint32]time.Duration `json:"games,omitempty"` // per game; empty when Steam gave only the total (2023)
}

// Year is one year's Replay.
type Year struct {
	Year     int                      `json:"year"`
	Played   time.Duration            `json:"played"`
	Sessions int                      `json:"sessions"`
	Games    map[uint32]time.Duration `json:"games"` // the year's time per game
	Months   []Month                  `json:"months"`
}

// Client fetches Replays from baseURL (the GetUserYearInReview method).
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for baseURL (config.ReplayURL).
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

// Fetch returns one year of the user's Replay.
func (c *Client) Fetch(ctx context.Context, steamID uint64, year int) (Year, error) {
	q := url.Values{"steamid": {strconv.FormatUint(steamID, 10)}, "year": {strconv.Itoa(year)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return Year{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Year{}, fmt.Errorf("replay %d: %w", year, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Year{}, fmt.Errorf("replay %d: HTTP %d", year, resp.StatusCode)
	}
	var body raw
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Year{}, fmt.Errorf("replay %d: %w", year, err)
	}
	return parse(body, year)
}

type rawStats struct {
	Seconds  uint32 `json:"total_playtime_seconds"`
	Sessions int    `json:"total_sessions"`
}

type rawGame struct {
	AppID uint32   `json:"appid"`
	Stats rawStats `json:"stats"`
}

type raw struct {
	Response struct {
		Stats *struct {
			Playtime struct {
				Total  rawStats  `json:"total_stats"`
				Games  []rawGame `json:"games"`
				Months []struct {
					Start uint32    `json:"rtime_month"`
					Stats rawStats  `json:"stats"`
					Games []rawGame `json:"appid"`
				} `json:"months"`
			} `json:"playtime_stats"`
		} `json:"stats"`
	} `json:"response"`
}

func seconds(s uint32) time.Duration { return time.Duration(s) * time.Second }

func parse(body raw, year int) (Year, error) {
	st := body.Response.Stats
	if st == nil || len(st.Playtime.Months) == 0 {
		return Year{}, ErrNotPublic
	}
	y := Year{Year: year, Played: seconds(st.Playtime.Total.Seconds), Sessions: st.Playtime.Total.Sessions, Games: map[uint32]time.Duration{}}
	for _, g := range st.Playtime.Games {
		y.Games[g.AppID] += seconds(g.Stats.Seconds)
	}
	for _, rm := range st.Playtime.Months {
		m := Month{Start: time.Unix(int64(rm.Start), 0).UTC(), Played: seconds(rm.Stats.Seconds), Sessions: rm.Stats.Sessions}
		if len(rm.Games) > 0 {
			m.Games = map[uint32]time.Duration{}
			for _, g := range rm.Games {
				m.Games[g.AppID] += seconds(g.Stats.Seconds)
			}
		}
		y.Months = append(y.Months, m)
	}
	return y, nil
}
