package steam

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Achievements is how many of a game's achievements the account unlocked.
type Achievements struct {
	Achieved int `json:"achieved"`
	Total    int `json:"total"`
}

// ReadAchievements reads achievement totals from the Steam client's cached
// library pages (userdata/<account>/config/librarycache/<appid>.json), which
// Steam writes for every game whose library page it has shown. Games without
// achievements, and files that can't be read, are left out.
func ReadAchievements(steamPath string, acct Account) (map[uint32]Achievements, error) {
	dir := filepath.Join(steamPath, "userdata", strconv.FormatUint(uint64(acct.AccountID), 10), "config", "librarycache")
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]Achievements)
	for _, f := range files {
		appID, err := strconv.ParseUint(strings.TrimSuffix(filepath.Base(f), ".json"), 10, 32)
		if err != nil || appID == 0 {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if a, ok := parseLibraryPage(b); ok {
			out[uint32(appID)] = a
		}
	}
	return out, nil
}

// parseLibraryPage finds the achievements section in one cached page: a JSON
// list of [name, body] pairs.
func parseLibraryPage(b []byte) (Achievements, bool) {
	var sections [][]json.RawMessage
	if err := json.Unmarshal(b, &sections); err != nil {
		return Achievements{}, false
	}
	for _, s := range sections {
		if len(s) != 2 {
			continue
		}
		var name string
		if err := json.Unmarshal(s[0], &name); err != nil || name != "achievements" {
			continue
		}
		var body struct {
			Data struct {
				Total    int `json:"nTotal"`
				Achieved int `json:"nAchieved"`
			} `json:"data"`
		}
		if err := json.Unmarshal(s[1], &body); err != nil || body.Data.Total <= 0 {
			return Achievements{}, false
		}
		a := Achievements{Achieved: min(max(body.Data.Achieved, 0), body.Data.Total), Total: body.Data.Total}
		return a, true
	}
	return Achievements{}, false
}
