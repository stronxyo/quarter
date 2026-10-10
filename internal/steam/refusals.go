package steam

// Games Steam refused to show in the library. When a steam://nav/games/
// details link names a game that isn't in the library, Steam's interface logs
// "SteamUI: ERROR: Unrecognized request details <id>" in logs/webhelper_js.txt
// (seen on Pablo's PC, 2026-10-01: every game it couldn't show, and none it
// could). Undocumented: when the log is missing or worded otherwise, nothing
// is found, and Quarter behaves as without it.

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

const (
	refusalLog  = "webhelper_js.txt"
	refusalTail = 8 << 20  // the log's last 8 MB at start: it rolls over at about 6 MB
	recentTail  = 64 << 10 // after a click: the last 64 KB is plenty
)

var reRefused = regexp.MustCompile(`^\[(\d{4}-\d\d-\d\d \d\d:\d\d:\d\d)\] SteamUI: ERROR: Unrecognized request details (\d+)`)

// LibraryRefusals reads every game Steam's log says it couldn't show in the
// library, with the last time it said so.
func LibraryRefusals(steamPath string, loc *time.Location) (map[uint32]time.Time, error) {
	return readRefusals(steamPath, refusalTail, loc)
}

// RefusedSince reports whether Steam's log says it couldn't show appID at
// or after since (the log has whole seconds).
func RefusedSince(steamPath string, appID uint32, since time.Time, loc *time.Location) bool {
	got, err := readRefusals(steamPath, recentTail, loc)
	if err != nil {
		return false
	}
	at, ok := got[appID]
	return ok && !at.Before(since.Truncate(time.Second))
}

func readRefusals(steamPath string, tail int64, loc *time.Location) (map[uint32]time.Time, error) {
	f, err := os.Open(filepath.Join(steamPath, "logs", refusalLog))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > tail {
		if _, err := f.Seek(-tail, io.SeekEnd); err != nil {
			return nil, err
		}
	}
	out := map[uint32]time.Time{}
	r := bufio.NewReader(f) // lines of any length: the interface's log has very long ones
	for {
		line, err := r.ReadString('\n')
		if m := reRefused.FindStringSubmatch(line); m != nil {
			at, err1 := time.ParseInLocation("2006-01-02 15:04:05", m[1], loc)
			id, err2 := strconv.ParseUint(m[2], 10, 32)
			if err1 == nil && err2 == nil && at.After(out[uint32(id)]) {
				out[uint32(id)] = at
			}
		}
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
	}
}
