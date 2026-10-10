package steam

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Session is one run of a game, as Steam logged it.
type Session struct {
	AppID    uint32    `json:"app"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`       // equals Start when EndKnown is false
	EndKnown bool      `json:"end_known"` // false: Steam stopped before the game did (a crash or a shutdown)
}

// Length is how long the session lasted; an unknown end counts no time.
func (s Session) Length() time.Duration {
	if !s.EndKnown {
		return 0
	}
	return s.End.Sub(s.Start)
}

// ReadSessions reads the finished game sessions in Steam's game process log
// (logs/gameprocess_log.txt, after its rotated .previous copy when there is
// one). The log's times are local to this PC, read in loc.
func ReadSessions(steamPath string, loc *time.Location) ([]Session, error) {
	logs := filepath.Join(steamPath, "logs")
	var r sessionReader
	read := 0
	for _, name := range []string{"gameprocess_log.previous.txt", "gameprocess_log.txt"} {
		f, err := os.Open(filepath.Join(logs, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		err = r.read(f, loc)
		f.Close()
		if err != nil {
			return nil, err
		}
		read++
	}
	if read == 0 {
		return nil, os.ErrNotExist
	}
	return r.done, nil
}

// sessionReader turns log lines into sessions. A session opens at a game's
// first "adding PID" line (a game may run several processes) and closes at
// "Remove <appid> from running list". A "Client version" line is Steam
// starting, so a session still open there ended with Steam, at an unknown
// time. Sessions still open at the end are games running now.
type sessionReader struct {
	open map[uint32]time.Time
	done []Session
}

func (r *sessionReader) read(f io.Reader, loc *time.Location) error {
	if r.open == nil {
		r.open = make(map[uint32]time.Time)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // launch lines carry whole command lines
	for sc.Scan() {
		r.line(strings.TrimRight(sc.Text(), "\r"), loc)
	}
	return sc.Err()
}

// logTime is the layout of a line's leading timestamp, inside brackets.
const logTime = "2006-01-02 15:04:05"

func (r *sessionReader) line(s string, loc *time.Location) {
	if len(s) < len(logTime)+3 || s[0] != '[' || s[len(logTime)+1] != ']' {
		return
	}
	at, err := time.ParseInLocation(logTime, s[1:len(logTime)+1], loc)
	if err != nil {
		return
	}
	rest := strings.TrimSpace(s[len(logTime)+2:])
	switch {
	case strings.HasPrefix(rest, "Client version:"):
		for id, start := range r.open {
			r.done = append(r.done, Session{AppID: id, Start: start, End: start})
		}
		clear(r.open)
	case strings.HasPrefix(rest, "AppID ") && strings.Contains(rest, " adding PID "):
		if id, ok := appIDField(strings.TrimPrefix(rest, "AppID ")); ok {
			if _, running := r.open[id]; !running {
				r.open[id] = at
			}
		}
	case strings.HasPrefix(rest, "Remove ") && strings.HasSuffix(rest, " from running list"):
		if id, ok := appIDField(strings.TrimPrefix(rest, "Remove ")); ok {
			if start, running := r.open[id]; running {
				r.done = append(r.done, Session{AppID: id, Start: start, End: at, EndKnown: true})
				delete(r.open, id)
			}
		}
	}
}

// appIDField reads the appid that starts s.
func appIDField(s string) (uint32, bool) {
	field, _, _ := strings.Cut(s, " ")
	id, err := strconv.ParseUint(field, 10, 32)
	return uint32(id), err == nil && id != 0
}
