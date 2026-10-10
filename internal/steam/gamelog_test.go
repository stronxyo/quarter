package steam

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A log in Steam's shape: CRLF lines, blank lines between client starts,
// games with helper processes, overlapping games, a crash and a stray end.
const sampleLog = "\r\n\r\n" +
	"[2026-03-21 20:42:28] Client version: 1773426488\r\n" +
	"[2026-03-21 20:42:28] SSGL: UI mode (0->7)\r\n" +
	// Game 10 with a helper process that exits first.
	"[2026-03-21 21:00:00] AppID 10 adding PID 100 as a tracked process \"\"C:\\Games\\ten.exe\"\"\r\n" +
	"[2026-03-21 21:00:05] AppID 10 adding PID 101 as a tracked process \"helper.exe\"\r\n" +
	"[2026-03-21 21:00:06] SSGL: change [10] NSCC 0->1\r\n" +
	// Game 20 starts while 10 runs.
	"[2026-03-21 21:30:00] AppID 20 adding PID 200 as a tracked process \"twenty.exe\"\r\n" +
	"[2026-03-21 21:59:59] AppID 10 no longer tracking PID 101, exit code 0\r\n" +
	"[2026-03-21 22:00:00] AppID 10 no longer tracking PID 100, exit code 0\r\n" +
	"[2026-03-21 22:00:00] Remove 10 from running list\r\n" +
	"[2026-03-21 22:10:00] Remove 20 from running list\r\n" +
	// An end with no start is ignored.
	"[2026-03-21 22:20:00] Remove 30 from running list\r\n" +
	// Game 40 is running when Steam goes down; the restart ends it.
	"[2026-03-22 10:00:00] AppID 40 adding PID 400 as a tracked process \"forty.exe\"\r\n" +
	"\r\n\r\n" +
	"[2026-03-23 09:00:00] Client version: 1773426488\r\n" +
	// Game 50 is still running at the end of the log.
	"[2026-03-23 09:30:00] AppID 50 adding PID 500 as a tracked process \"fifty.exe\"\r\n" +
	"garbage line\r\n" +
	"[bad time] Remove 50 from running list\r\n"

func sorted(s []Session) []Session {
	slices.SortFunc(s, func(a, b Session) int { return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.AppID, b.AppID)) })
	return s
}

func at(s string) time.Time {
	t, err := time.ParseInLocation(logTime, s, time.UTC)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSessionReader(t *testing.T) {
	var r sessionReader
	if err := r.read(strings.NewReader(sampleLog), time.UTC); err != nil {
		t.Fatal(err)
	}
	got := sorted(r.done)
	want := []Session{
		{AppID: 10, Start: at("2026-03-21 21:00:00"), End: at("2026-03-21 22:00:00"), EndKnown: true},
		{AppID: 20, Start: at("2026-03-21 21:30:00"), End: at("2026-03-21 22:10:00"), EndKnown: true},
		{AppID: 40, Start: at("2026-03-22 10:00:00"), End: at("2026-03-22 10:00:00")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	if got[2].Length() != 0 {
		t.Error("a session with an unknown end counts no time")
	}
	if got[0].Length() != time.Hour {
		t.Errorf("length: got %v, want 1h", got[0].Length())
	}
}

func TestReadSessionsLocalTime(t *testing.T) {
	steam := t.TempDir()
	writeLog(t, steam, "gameprocess_log.txt", sampleLog)
	loc := time.FixedZone("UTC-3", -3*60*60)
	got, err := ReadSessions(steam, loc)
	if err != nil {
		t.Fatal(err)
	}
	got = sorted(got)
	if len(got) != 3 {
		t.Fatalf("got %d sessions, want 3", len(got))
	}
	if want := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC); !got[0].Start.Equal(want) {
		t.Errorf("21:00 at UTC-3 is %v, got %v", want, got[0].Start)
	}
}

// A session that began before the log rotated ends in the new log.
func TestReadSessionsRotated(t *testing.T) {
	steam := t.TempDir()
	writeLog(t, steam, "gameprocess_log.previous.txt",
		"[2026-03-21 21:00:00] AppID 10 adding PID 100 as a tracked process \"ten.exe\"\r\n")
	writeLog(t, steam, "gameprocess_log.txt",
		"[2026-03-21 22:00:00] Remove 10 from running list\r\n")
	got, err := ReadSessions(steam, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{AppID: 10, Start: at("2026-03-21 21:00:00"), End: at("2026-03-21 22:00:00"), EndKnown: true}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestReadSessionsNoLog(t *testing.T) {
	if _, err := ReadSessions(t.TempDir(), time.UTC); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got %v, want ErrNotExist", err)
	}
}

func writeLog(t *testing.T, steam, name, text string) {
	t.Helper()
	dir := filepath.Join(steam, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
