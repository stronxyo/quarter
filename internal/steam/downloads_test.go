package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A download in the shape Steam logs it (from DiRT Rally 2.0's install,
// 2026-09-29, trimmed), with a rate from before it started.
const oneDownload = "[2026-09-29 23:40:00] Current download rate: 999.000 Mbps\r\n" +
	"[2026-09-29 23:41:08] AppID 690790 update started : download 1000000000/10000000000, store 0/0, reuse 0/0, delta 0/0, stage 0/10000000000 \r\n" +
	"[2026-09-29 23:41:08] AppID 690790 App update changed : Running Update,Downloading,Staging,\r\n" +
	"[2026-09-29 23:42:08] Current download rate: 400.000 Mbps\r\n" + // 50 MB/s for 60 s: +3 GB
	"[2026-09-29 23:43:08] Current download rate: 400.000 Mbps\r\n" // +3 GB: 7 GB

func TestParseDownloads(t *testing.T) {
	d, ok := parseDownloads(strings.NewReader(oneDownload), time.UTC)[690790]
	if !ok {
		t.Fatal("the download should be found")
	}
	if d.Phase != PhaseDownloading || d.Total != 10e9 || d.Done != 7e9 || d.Rate != 50e6 || !d.At.Equal(at("2026-09-29 23:43:08")) {
		t.Errorf("got %+v, want downloading 7 of 10 GB at 50 MB/s, as of 23:43:08", d)
	}
}

func TestParseDownloadsPhasesAndEnd(t *testing.T) {
	applying := oneDownload + "[2026-09-29 23:44:00] AppID 690790 App update changed : Running Update,Staging,\r\n"
	if d := parseDownloads(strings.NewReader(applying), time.UTC)[690790]; d.Phase != PhaseApplying || d.Rate != 0 {
		t.Errorf("staging after the download is applying, with no rate: %+v", d)
	}
	for _, end := range []string{
		"[2026-09-29 23:45:00] AppID 690790 finished update, 30 mounted depots (BuildID 6444206)\r\n",
		"[2026-09-29 23:45:00] AppID 690790 scheduler finished : removed from schedule (result No Error, state 0xc)\r\n",
	} {
		if got := parseDownloads(strings.NewReader(applying+end), time.UTC); len(got) != 0 {
			t.Errorf("after %q nothing is downloading: %+v", end[22:50], got)
		}
	}
}

// A pause ("None") stops the count; the time paused doesn't count after it
// resumes.
func TestParseDownloadsPause(t *testing.T) {
	log := oneDownload +
		"[2026-09-29 23:44:00] AppID 690790 App update changed : None\r\n" +
		"[2026-09-29 23:50:00] Current download rate: 0.000 Mbps\r\n" +
		"[2026-09-29 23:55:00] AppID 690790 App update changed : Running Update,Downloading,Staging,\r\n" +
		"[2026-09-29 23:56:00] Current download rate: 400.000 Mbps\r\n" // +3 GB since the resume
	d := parseDownloads(strings.NewReader(log), time.UTC)[690790]
	if d.Done != 10e9 || d.Phase != PhaseDownloading {
		t.Errorf("got %d bytes, want 10 GB: only the minute after resuming counts", d.Done)
	}
}

// While two games are marked downloading the rate can't be split: neither
// advances.
func TestParseDownloadsTwoAtOnce(t *testing.T) {
	log := oneDownload +
		"[2026-09-29 23:43:30] AppID 620 update started : download 0/5000000000, store 0/0\r\n" +
		"[2026-09-29 23:43:30] AppID 620 App update changed : Running Update,Downloading,\r\n" +
		"[2026-09-29 23:44:30] Current download rate: 400.000 Mbps\r\n"
	got := parseDownloads(strings.NewReader(log), time.UTC)
	if got[690790].Done != 7e9 || got[620].Done != 0 {
		t.Errorf("neither should advance: %d and %d", got[690790].Done, got[620].Done)
	}
}

// A Workshop update (a game's mods) logs its own changes and ends with
// "scheduler finished" only; its "update started" must not linger.
func TestParseDownloadsWorkshop(t *testing.T) {
	log := "[2026-09-29 20:14:06] AppID 241100 update started : download 0/7696, store 0/0\r\n" +
		"[2026-09-29 20:14:07] AppID 241100 Workshop update changed : Running Update,Staging,\r\n" +
		"[2026-09-29 20:14:07] AppID 241100 scheduler finished : removed from schedule (result No Error, state 0x0)\r\n"
	if got := parseDownloads(strings.NewReader(log), time.UTC); len(got) != 0 {
		t.Errorf("the Workshop update is over: %+v", got)
	}
}

func TestEstimate(t *testing.T) {
	d := Download{Phase: PhaseDownloading, Done: 7e9, Total: 20e9, Rate: 50e6, At: at("2026-09-29 23:43:08")}
	tests := []struct {
		now  string
		want int64
	}{
		{"2026-09-29 23:43:08", 7e9},
		{"2026-09-29 23:43:28", 7e9 + 20*50e6}, // carried on 20 s
		{"2026-09-29 23:50:00", 7e9 + 90*50e6}, // no further than 90 s past the log
	}
	for _, tt := range tests {
		if got := d.Estimate(at(tt.now)); got != tt.want {
			t.Errorf("at %s: got %d, want %d", tt.now, got, tt.want)
		}
	}
	d.Done = 19.9e9
	if got := d.Estimate(at("2026-09-29 23:44:00")); got != d.Total {
		t.Errorf("never past the total: got %d", got)
	}
	d.Phase = PhaseApplying
	if got := d.Estimate(at("2026-09-29 23:44:00")); got != 19.9e9 {
		t.Errorf("only a download carries on: got %d", got)
	}
}

func TestReadDownloadsFile(t *testing.T) {
	steam := t.TempDir()
	if err := os.MkdirAll(filepath.Join(steam, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steam, "logs", "content_log.txt"), []byte(oneDownload), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadDownloads(steam, time.UTC)
	if err != nil || got[690790].Done != 7e9 {
		t.Errorf("got %+v, %v", got, err)
	}
	if _, err := ReadDownloads(t.TempDir(), time.UTC); err == nil {
		t.Error("no log is an error")
	}
}

func TestManifestDownloadState(t *testing.T) {
	tests := []struct {
		flags, build  uint32
		first, paused bool
	}{
		{1026, 0, true, false},        // a first install
		{1026, 6444206, false, false}, // an update
		{1026 | 512, 0, true, true},   // a first install, paused
		{4, 0, false, false},          // installed
	}
	for _, tt := range tests {
		m := Manifest{StateFlags: tt.flags, BuildID: tt.build}
		if m.FirstInstall() != tt.first || m.Paused() != tt.paused {
			t.Errorf("flags %d build %d: first %v paused %v, want %v %v", tt.flags, tt.build, m.FirstInstall(), m.Paused(), tt.first, tt.paused)
		}
	}
	lib := t.TempDir()
	write(t, lib, "steamapps/appmanifest_690790.acf", `"AppState" { "appid" "690790" "name" "DiRT Rally 2.0" "StateFlags" "1026" "buildid" "0" }`)
	m, err := ReadManifest(lib, 690790)
	if err != nil || !m.FirstInstall() || m.Name != "DiRT Rally 2.0" {
		t.Errorf("ReadManifest: %+v, %v", m, err)
	}
}
