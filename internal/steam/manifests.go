package steam

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/stronxyo/quarter/internal/vdf"
)

// stateFullyInstalled is the StateFlags bit Steam sets when a game is
// installed, up to date and playable.
const stateFullyInstalled = 4

// Manifest is one appmanifest_<appid>.acf: a game Steam knows is on disk.
type Manifest struct {
	AppID       uint32
	Name        string
	InstallDir  string
	LibraryPath string
	SizeOnDisk  int64
	StateFlags  uint32
	LastUpdated time.Time
}

// Installed reports whether Steam considers the game fully installed.
func (m Manifest) Installed() bool {
	return m.StateFlags&stateFullyInstalled != 0
}

// InstalledGames scans every library for fully installed games, dropping
// Steam tools. Libraries that aren't connected (an unplugged drive) are
// returned in missing; files that can't be read are skipped and returned as
// warnings.
func InstalledGames(steamPath string) (games []Manifest, missing []LibraryFolder, warnings []error) {
	libs, err := LibraryFolders(steamPath)
	if err != nil {
		warnings = append(warnings, err)
	}
	for _, lib := range libs {
		if !lib.connected() {
			missing = append(missing, lib)
			continue
		}
		manifests, errs := Manifests(lib.Path)
		warnings = append(warnings, errs...)
		for _, m := range manifests {
			if m.Installed() && IsGame(m.AppID) {
				games = append(games, m)
			}
		}
	}
	return games, missing, warnings
}

// Manifests reads every appmanifest in one library, sorted by appid.
func Manifests(library string) ([]Manifest, []error) {
	files, err := filepath.Glob(filepath.Join(library, "steamapps", "appmanifest_*.acf"))
	if err != nil {
		return nil, []error{err}
	}

	var manifests []Manifest
	var warnings []error
	for _, file := range files {
		root, err := readVDF(file)
		if err != nil {
			warnings = append(warnings, err)
			continue
		}
		m, err := parseManifest(root, library)
		if err != nil {
			warnings = append(warnings, &FileError{Name: filepath.Base(file), Err: err})
			continue
		}
		manifests = append(manifests, m)
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].AppID < manifests[j].AppID })
	return manifests, warnings
}

func parseManifest(root *vdf.Node, library string) (Manifest, error) {
	state := root.Get("AppState")
	if state == nil {
		return Manifest{}, errors.New("no AppState block")
	}
	appID, err := strconv.ParseUint(state.ValueOf("appid"), 10, 32)
	if err != nil || appID == 0 {
		return Manifest{}, errors.New("missing or invalid appid")
	}

	m := Manifest{
		AppID:       uint32(appID),
		Name:        state.ValueOf("name"),
		InstallDir:  state.ValueOf("installdir"),
		LibraryPath: library,
		SizeOnDisk:  parseInt(state.ValueOf("SizeOnDisk")),
		StateFlags:  uint32(parseInt(state.ValueOf("StateFlags"))),
	}
	if ts := parseInt(state.ValueOf("LastUpdated")); ts > 0 {
		m.LastUpdated = time.Unix(ts, 0)
	}
	if m.Name == "" {
		m.Name = fmt.Sprintf("App %d", m.AppID)
	}
	return m, nil
}

// parseInt reads a non-negative number, treating anything unreadable as 0.
func parseInt(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
