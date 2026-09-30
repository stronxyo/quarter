package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// dlNow is on a whole second, when the arrow shows; half a second later it
// is off.
var dlNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local)

// Bravo 4 of 10 GB in, as of dlNow, at 10 MB/s.
var bravoDownload = steam.Download{AppID: 2, Phase: steam.PhaseDownloading, Done: 4e9, Total: 10e9, Rate: 10e6, At: dlNow}

func dlModel(t *testing.T, d map[uint32]steam.Download, games ...library.Game) Model {
	t.Helper()
	if games == nil {
		games = updatingGames
	}
	m, _ := withOpener(t, nil, games...)
	m.now = func() time.Time { return dlNow }
	m.downloads = d
	m.list.SetDelegate(m.delegate())
	return m
}

func TestDownloadShown(t *testing.T) {
	m := dlModel(t, map[uint32]steam.Download{2: bravoDownload})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, _ = tm.Update(press("j")) // select Bravo
	v := runningView(tm)
	if line := lineWith(v, "Bravo"); !strings.Contains(line, "↓ ~40%") {
		t.Errorf("Bravo's row should show its progress: %q", line)
	}
	for _, want := range []string{"↓ UPDATING ~40%", "━━━━━━━━━━━━━━━━────────────────────────", "4.0 GB of 10.0 GB · 10 MB/s · about 10 min left"} {
		if !strings.Contains(v, want) {
			t.Errorf("the pane should show %q:\n%s", want, v)
		}
	}
}

// Every state in the row and the pane; the row always fits the column.
func TestDownloadStates(t *testing.T) {
	bravo := updatingGames[1]
	installing := bravo
	installing.Installing = true
	applying := bravoDownload
	applying.Phase = steam.PhaseApplying
	verifying := bravoDownload
	verifying.Phase = steam.PhaseVerifying
	other := map[uint32]steam.Download{9: {AppID: 9, Phase: steam.PhaseDownloading}}

	tests := []struct {
		name        string
		g           library.Game
		d           map[uint32]steam.Download
		paused      bool
		cell, title string
		info        string
	}{
		{"downloading", bravo, map[uint32]steam.Download{2: bravoDownload}, false, "↓ ~40%", "↓ UPDATING ~40%", "4.0 GB of 10.0 GB"},
		{"installing", installing, map[uint32]steam.Download{2: bravoDownload}, false, "↓ ~40%", "↓ INSTALLING ~40%", "4.0 GB of 10.0 GB"},
		{"before the log", installing, nil, false, "↓ installing", "↓ INSTALLING", "Steam is getting it ready."},
		{"applying", bravo, map[uint32]steam.Download{2: applying}, false, "↓ applying", "↓ UPDATING · APPLYING", "putting the files in place"},
		{"verifying", bravo, map[uint32]steam.Download{2: verifying}, false, "↓ verifying", "↓ UPDATING · VERIFYING", "checking the files"},
		{"paused keeps its progress", bravo, map[uint32]steam.Download{2: bravoDownload}, true, "↓ paused", "↓ UPDATING ~40% · PAUSED", "Paused in Steam."},
		{"queued", bravo, other, false, "↓ queued", "↓ UPDATING · QUEUED", "download queue"},
	}
	for _, tt := range tests {
		v := dlView{downloads: tt.d, now: func() time.Time { return dlNow }}
		if tt.paused {
			v.paused = map[uint32]bool{2: true}
		}
		cell := v.cell(tt.g)
		if cell != tt.cell || ansi.StringWidth(cell) > agoWidth {
			t.Errorf("%s: row %q, want %q, within %d", tt.name, cell, tt.cell, agoWidth)
		}
		lines := v.lines(tt.g, 60)
		if got := ansi.Strip(lines[0]); got != tt.title {
			t.Errorf("%s: title %q, want %q", tt.name, got, tt.title)
		}
		if got := ansi.Strip(lines[len(lines)-1]); !strings.Contains(got, tt.info) {
			t.Errorf("%s: info %q, want %q", tt.name, got, tt.info)
		}
	}
}

// The arrow blinks while Steam works on the download, and holds still while
// it's paused. The estimate carries on between the log's lines.
func TestDownloadBlink(t *testing.T) {
	g := updatingGames[1]
	half := dlNow.Add(blinkStep)
	v := dlView{downloads: map[uint32]steam.Download{2: bravoDownload}, now: func() time.Time { return half }}
	if got := v.cell(g); got != "  ~40%" {
		t.Errorf("half a second on, the arrow is off: %q", got)
	}
	v.paused = map[uint32]bool{2: true}
	if got := v.cell(g); got != "↓ paused" {
		t.Errorf("a paused download doesn't blink: %q", got)
	}
	later := dlNow.Add(60 * time.Second) // +600 MB
	v = dlView{downloads: map[uint32]steam.Download{2: bravoDownload}, now: func() time.Time { return later }}
	if got := v.cell(g); got != "↓ ~46%" {
		t.Errorf("a minute on: %q", got)
	}
}

// The blink runs while a game downloads, one tick at a time, and stops after.
func TestDownloadBlinkTicks(t *testing.T) {
	m := dlModel(t, nil)
	if !m.dlBlink {
		t.Fatal("a downloading game should start the blink")
	}
	if m.blinkDownloads() != nil {
		t.Error("only one blink runs at a time")
	}
	if _, cmd := m.dlBlinkStep(); cmd == nil {
		t.Error("the blink goes on while Bravo downloads")
	}
	m = dlModel(t, nil, updatingGames[0])
	m.dlBlink = true
	if m, cmd := m.dlBlinkStep(); cmd != nil || m.dlBlink {
		t.Error("nothing downloading: the blink stops")
	}
}

// A download that ended, or one the library didn't know as downloading, reads
// the library again; one seen before doesn't, even for an app the library
// never lists.
func TestDownloadChangesReadLibrary(t *testing.T) {
	redist := steam.Download{AppID: 228980, Phase: steam.PhaseDownloading}
	alpha := steam.Download{AppID: 1, Phase: steam.PhaseDownloading}
	tests := []struct {
		name       string
		before, d  map[uint32]steam.Download
		wantReread bool
	}{
		{"Bravo ended", map[uint32]steam.Download{2: bravoDownload}, map[uint32]steam.Download{}, true},
		{"Alpha started", map[uint32]steam.Download{2: bravoDownload}, map[uint32]steam.Download{1: alpha, 2: bravoDownload}, true},
		{"an unlisted app started", nil, map[uint32]steam.Download{228980: redist}, true},
		{"the unlisted app goes on", map[uint32]steam.Download{228980: redist}, map[uint32]steam.Download{228980: redist}, false},
		{"Bravo goes on", map[uint32]steam.Download{2: bravoDownload}, map[uint32]steam.Download{2: bravoDownload}, false},
	}
	for _, tt := range tests {
		m := dlModel(t, tt.before)
		m.scanning = false
		m, _ = m.applyDownloads(downloadsMsg{reread: true, d: tt.d, size: 100})
		if m.scanning != tt.wantReread {
			t.Errorf("%s: library read %v, want %v", tt.name, m.scanning, tt.wantReread)
		}
		if m.logSize != 100 {
			t.Errorf("%s: the log's size should be kept", tt.name)
		}
	}
}

// The log is read only when its size changed; the downloading games' install
// files are read at every look, for a pause.
func TestCheckDownloads(t *testing.T) {
	root, lib := t.TempDir(), t.TempDir()
	log := "[2026-09-30 11:59:00] AppID 2 update started : download 0/10000000000, store 0/0\r\n" +
		"[2026-09-30 11:59:00] AppID 2 App update changed : Running Update,Downloading,\r\n"
	writeFile(t, filepath.Join(root, "logs", "content_log.txt"), log)
	writeFile(t, filepath.Join(lib, "steamapps", "appmanifest_2.acf"), `"AppState" { "appid" "2" "StateFlags" "1538" "buildid" "7" }`)

	bravo := updatingGames[1]
	bravo.LibraryPath = lib
	m := dlModel(t, nil, updatingGames[0], bravo)
	m.steamRoot = root

	msg := m.checkDownloads()().(downloadsMsg)
	if !msg.reread || msg.size != int64(len(log)) || msg.d[2].Total != 10e9 {
		t.Fatalf("the first look reads the log: %+v", msg)
	}
	if !msg.paused[2] {
		t.Error("Bravo's install file says paused")
	}
	m.logSize = msg.size
	if msg := m.checkDownloads()().(downloadsMsg); msg.reread || !msg.paused[2] {
		t.Errorf("the same size: no read of the log, the install file still: %+v", msg)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
