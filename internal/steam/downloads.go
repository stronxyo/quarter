package steam

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Steam writes a game's install file (appmanifest) only when a download
// starts, pauses or ends, so live progress comes from its download log,
// logs/content_log.txt (seen 2026-09-29 on DiRT Rally 2.0):
//
//	[2026-09-29 23:41:08] AppID 690790 update started : download 10452917008/115175262656, ...
//	[2026-09-29 23:41:08] AppID 690790 App update changed : Running Update,Downloading,Staging,
//	[2026-09-29 23:42:21] Current download rate: 524.432 Mbps
//	[2026-09-30 00:10:54] AppID 690790 finished update, 30 mounted depots ...
//
// The rate is the client's total, written about once a minute. Adding it up
// over the download from its starting bytes came to 104.1 GB against the
// real 104.7 GB (99%). Steam downloads one game at a time; while two are
// marked downloading, the rate can't be split and neither advances.

// DownloadPhase is where a download is, as the log says.
type DownloadPhase int

const (
	PhaseNone        DownloadPhase = iota // not in the log's current run: queued, paused or done
	PhaseDownloading                      // bytes coming in
	PhaseApplying                         // staging or committing the downloaded files
	PhaseVerifying                        // checking the files
)

// Download is one game's download as Steam's log tells it.
type Download struct {
	AppID uint32
	Phase DownloadPhase
	Done  int64     // bytes downloaded at At, an estimate from the log's rates
	Total int64     // bytes to download
	Rate  float64   // bytes per second, the client's last reading; 0 when unknown
	At    time.Time // when Done was reached; after it Done grows at Rate
}

// maxCarry is how long past the log's last line an estimate carries on at
// its speed: Steam logs the rate about once a minute, and a stalled
// download mustn't count itself up.
const maxCarry = 90 * time.Second

// Estimate is the bytes downloaded by now: Done, carried on at Rate since
// At while downloading (for at most maxCarry), never past Total.
func (d Download) Estimate(now time.Time) int64 {
	done := d.Done
	if d.Phase == PhaseDownloading && d.Rate > 0 && now.After(d.At) {
		done += int64(d.Rate * min(now.Sub(d.At), maxCarry).Seconds())
	}
	return min(max(done, 0), d.Total)
}

// logTail is how much of the end of the download log is read: a long
// download's lines fit in far less (a line a minute).
const logTail = 4 << 20

var (
	reStarted = regexp.MustCompile(`^\[([^\]]+)\] AppID (\d+) update started : download (\d+)/(\d+)`)
	reChanged = regexp.MustCompile(`^\[([^\]]+)\] AppID (\d+) App update changed : (.*)$`)
	reRate    = regexp.MustCompile(`^\[([^\]]+)\] Current download rate: ([\d.]+) Mbps`)
	// A download ends with "finished update" and "scheduler finished"; a
	// Workshop update (a game's mods) logs only the second.
	reFinished = regexp.MustCompile(`^\[([^\]]+)\] AppID (\d+) (?:finished update|scheduler finished)`)
)

// ReadDownloads reads the games Steam's download log shows in progress. Its
// times are local to this PC, read in loc.
func ReadDownloads(steamPath string, loc *time.Location) (map[uint32]Download, error) {
	f, err := os.Open(filepath.Join(steamPath, "logs", "content_log.txt"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > logTail {
		if _, err := f.Seek(-logTail, io.SeekEnd); err != nil {
			return nil, err
		}
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return parseDownloads(bytes.NewReader(b), loc), nil
}

func parseDownloads(r io.Reader, loc *time.Location) map[uint32]Download {
	type state struct {
		d    Download
		last time.Time // the last time Done was brought up to date
	}
	apps := map[uint32]*state{}
	var rate float64
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	at := func(s string) (time.Time, bool) {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
		return t, err == nil
	}
	id := func(s string) uint32 {
		n, _ := strconv.ParseUint(s, 10, 32)
		return uint32(n)
	}
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if m := reRate.FindStringSubmatch(line); m != nil {
			t, ok := at(m[1])
			if !ok {
				continue
			}
			mbps, _ := strconv.ParseFloat(m[2], 64)
			rate = mbps * 1e6 / 8
			var downloading []*state
			for _, s := range apps {
				if s.d.Phase == PhaseDownloading {
					downloading = append(downloading, s)
				}
			}
			if len(downloading) == 1 {
				s := downloading[0]
				if t.After(s.last) {
					s.d.Done += int64(rate * t.Sub(s.last).Seconds())
					s.last = t
				}
			}
			continue
		}
		if m := reStarted.FindStringSubmatch(line); m != nil {
			t, ok := at(m[1])
			if !ok {
				continue
			}
			done, _ := strconv.ParseInt(m[3], 10, 64)
			total, _ := strconv.ParseInt(m[4], 10, 64)
			apps[id(m[2])] = &state{d: Download{AppID: id(m[2]), Done: done, Total: total}, last: t}
			continue
		}
		if m := reChanged.FindStringSubmatch(line); m != nil {
			s := apps[id(m[2])]
			if s == nil {
				continue
			}
			t, _ := at(m[1])
			was := s.d.Phase
			switch flags := m[3]; {
			case strings.Contains(flags, "Downloading"):
				s.d.Phase = PhaseDownloading
			case strings.Contains(flags, "Verifying"):
				s.d.Phase = PhaseVerifying
			case strings.Contains(flags, "Staging"), strings.Contains(flags, "Committing"):
				s.d.Phase = PhaseApplying
			default:
				s.d.Phase = PhaseNone
			}
			if s.d.Phase == PhaseDownloading && was != PhaseDownloading {
				s.last = t // time paused or queued doesn't count
			}
			continue
		}
		if m := reFinished.FindStringSubmatch(line); m != nil {
			delete(apps, id(m[2]))
		}
	}
	out := make(map[uint32]Download, len(apps))
	for app, s := range apps {
		d := s.d
		d.Done = min(d.Done, d.Total)
		d.At = s.last
		if d.Phase == PhaseDownloading {
			d.Rate = rate
		}
		out[app] = d
	}
	return out
}
