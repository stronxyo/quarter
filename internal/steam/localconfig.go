package steam

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/stronxyo/quarter/internal/vdf"
)

// PlayStats is what Steam records locally about playing one game.
type PlayStats struct {
	Playtime   time.Duration
	LastPlayed time.Time // zero if never played
}

// ReadPlayStats reads playtime and last played for every app the account has
// data for, keyed by appid.
func ReadPlayStats(steamPath string, acct Account) (map[uint32]PlayStats, error) {
	file := filepath.Join(steamPath, "userdata", strconv.FormatUint(uint64(acct.AccountID), 10), "config", "localconfig.vdf")
	root, err := readVDF(file)
	if err != nil {
		return nil, err
	}
	return parsePlayStats(root), nil
}

func parsePlayStats(root *vdf.Node) map[uint32]PlayStats {
	apps := root.Path("UserLocalConfigStore", "Software", "Valve", "Steam", "apps").Entries()
	stats := make(map[uint32]PlayStats, len(apps))
	for _, app := range apps {
		appID, err := strconv.ParseUint(app.Key, 10, 32)
		if err != nil || appID == 0 {
			continue
		}
		var s PlayStats
		s.Playtime = time.Duration(parseInt(app.ValueOf("Playtime"))) * time.Minute
		if ts := parseInt(app.ValueOf("LastPlayed")); ts > 0 {
			s.LastPlayed = time.Unix(ts, 0)
		}
		if s.Playtime > 0 || !s.LastPlayed.IsZero() {
			stats[uint32(appID)] = s
		}
	}
	return stats
}
