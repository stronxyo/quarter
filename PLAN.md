# Quarter: v0.1 Implementation Plan

Companion to `PROJECT_BRIEF.md`. Structure first, then milestones. Every milestone ends with something runnable on Windows.

## Project structure

```
quarter/
├─ cmd/quarter/main.go        # flags, wiring, starts the TUI (or --list plain output)
├─ internal/
│  ├─ vdf/                    # text VDF/ACF parser (+ tests)
│  ├─ steam/
│  │  ├─ locate_windows.go    # SteamPath from registry (golang.org/x/sys/windows/registry)
│  │  ├─ locate_other.go      # ~/.steam/steam, ~/Library/Application Support/Steam
│  │  ├─ libraries.go         # libraryfolders.vdf → library paths
│  │  ├─ manifests.go         # appmanifest_*.acf → installed apps
│  │  ├─ users.go             # loginusers.vdf → SteamID64 / accountid
│  │  ├─ localconfig.go       # localconfig.vdf → playtime, last played
│  │  ├─ filter.go            # non-game appid denylist
│  │  └─ launch.go            # steam:// URIs, opened via the OS handler
│  ├─ webapi/
│  │  ├─ client.go            # GetOwnedGames, GetPlayerAchievements
│  │  └─ store.go             # appdetails + rate limiter
│  ├─ library/
│  │  ├─ game.go              # Game model (merged view of all sources)
│  │  └─ merge.go             # local + API + store + favourites → []Game
│  ├─ cache/                  # JSON cache files + image cache, with timestamps
│  ├─ config/                 # paths (%APPDATA%\quarter), config.json, favourites
│  ├─ art/
│  │  ├─ source.go            # librarycache → disk cache → CDN
│  │  └─ halfblock.go         # image → "▀" truecolor string
│  └─ ui/
│     ├─ app.go               # root Bubbletea model, messages, layout
│     ├─ list.go              # library pane: search, sort, filter
│     ├─ detail.go            # detail pane
│     ├─ statusbar.go
│     ├─ help.go              # ? overlay
│     ├─ wizard.go            # first-run / API key setup
│     ├─ keys.go              # keymap (bubbles/key)
│     └─ styles.go            # Lip Gloss styles
└─ testdata/steam/            # fake Steam install: libraryfolders, manifests, loginusers, localconfig
```

Key decisions:
- **`internal/steam` knows nothing about the UI**, and `ui` never reads files directly. Data comes in as Bubbletea messages from background commands, so the UI never blocks.
- **`--steam-path` flag / `QUARTER_STEAM_PATH` env** overrides detection. This is what makes the scanner testable against `testdata/steam/` and debuggable on odd installs.
- **`quarter --list`** prints the scanned library as plain text. Useful for M1 and for bug reports forever after.
- **Launching** on Windows uses `rundll32 url.dll,FileProtocolHandler <uri>` (no console window flash, no `cmd` quoting issues). `open` on macOS, `xdg-open` on Linux.
- **Every local-format parser fails soft**: a bad `localconfig.vdf` means no playtime, not a crash.

### Game model (first cut)

```go
type Game struct {
    AppID        uint32
    Name         string
    Installed    bool
    LibraryPath  string
    SizeOnDisk   int64
    Playtime     time.Duration  // localconfig, overridden by API when present
    LastPlayed   time.Time
    Favourite    bool
    Achievements *Progress      // nil until fetched
    Meta         *StoreMeta     // nil until fetched
}
```

## Batches

Workflow per batch (see `RULES/QUARTER_RULES_V1.md` section 6): spec in `specs/BATCH_XX_SPEC.md`, Pablo approves, branch `batch/NN-slug` from `main`, one commit per phase, PR to `main`, Pablo merges.

### Batch 0: Setup and verification (`batch/00-setup`)
- **Phase A, toolchain:** install Go (`winget install GoLang.Go`), `go mod init github.com/stronxyo/quarter`, add Bubbletea, Bubbles, Lip Gloss, sahilm/fuzzy, pin versions.
- **Phase B, skeleton:** folder structure from the rules, `cmd/quarter/main.go` with `--version`, `internal/config/constants.go`.
- **Phase C, verify claims:** research every item in Rules Appendix C, with Sources. Update the brief and rules with the results.
- Done when: `go run ./cmd/quarter --version` prints a version; Appendix C is empty or each item is marked verified/wrong.

### Batch 1: Scan (`batch/01-scan`)
- **Phase A, VDF parser:** text VDF/ACF parser + table-driven tests.
- **Phase B, locate and libraries:** Steam path (registry + `--steam-path` / `QUARTER_STEAM_PATH`), `libraryfolders.vdf`, `appmanifest_*.acf`, non-game denylist.
- **Phase C, user data:** `loginusers.vdf` (SteamID64), `localconfig.vdf` (playtime, last played).
- **Phase D, `--list`:** plain-text library output.
- Done when: `quarter --list` prints your real installed games with playtime; all parsers tested against `testdata/`.

### Batch 2: List UI (`batch/02-list-ui`)
- **Phase A, list:** library pane, `j/k`, status bar, `q`.
- **Phase B, search/sort/filter:** fuzzy search (`/`), sort (`o`), filter (`Tab`).
- **Phase C, cache:** library index cached to JSON; cache shown instantly, rescan in background.
- **Phase D, benchmarks:** startup (cache to interactive list, target under 100 ms) and search on a 5,000-game fixture.
- Done when: you browse and search your real library in Windows Terminal; both benchmarks pass.

### Batch 3: Actions (`batch/03-actions`)
- **Phase A, launch:** `Enter` play, `i` install, `s` store page via `steam://` URIs.
- **Phase B, favourites and refresh:** `f` (persisted in config), `r`.
- Done when: you launch a game from Quarter.

### Batch 4: Sixel spike (`batch/04-sixel-spike`, time-boxed to half a day)
- **Phase A, prototype:** one sixel image beside a Bubbletea list; move selection; resize the window.
- **Phase B, decision:** sixel as opt-in in v0.1, or half-block only. Record in the rules (Appendix A) and the brief.
- Done when: the decision is written down. Prototype code merges only if the decision is "yes".

### Batch 5: Detail pane (`batch/05-detail`)
- **Phase A, layout:** two panes; collapse to overlay on narrow terminals.
- **Phase B, store metadata:** `appdetails` for the selected game, rate limiter, cache (refresh window: days).
- Done when: selecting a game shows developer, year, genres, description, playtime.

### Batch 6: Web API (`batch/06-webapi`)
- **Phase A, setup:** API key wizard (SteamID64 auto-detected), plain-text storage notice.
- **Phase B, owned games:** `GetOwnedGames` merges uninstalled games; private-profile detection and message.
- **Phase C, achievements:** fetched lazily for the selected game, progress bar.
- Done when: uninstalled games appear with `○`; achievements show in the detail pane.

### Batch 7: Cover art (`batch/07-art`)
- **Phase A, sources:** `librarycache` to disk cache to CDN.
- **Phase B, render:** half-block renderer (plus sixel if Batch 4 said yes).
- Done when: cover art shows in the detail pane.

### Batch 8: Polish and release (`batch/08-release`)
- **Phase A, UX:** `?` help overlay, "INSERT COIN" first-run splash, "GAME OVER" on quit.
- **Phase B, error states:** Steam not found, no libraries, offline.
- **Phase C, performance:** re-check the brief's targets; `pprof` any misses.
- **Phase D, release:** GoReleaser (Windows/macOS/Linux, amd64 + arm64, stripped binaries); smoke test on a clean terminal and, if available, a Steam Deck.
- Done when: v0.1 tagged.

## Testing

- Unit tests for `vdf`, `steam/*` parsers, `filter`, `library/merge`, run against `testdata/`
- `webapi` tested with `httptest` servers, never the real API in tests
- UI: key flows checked with `teatest` (search narrows list, filter cycles, Enter emits the launch URI) where cheap; the rest by hand in Windows Terminal

## Open questions

1. ~~Go module path~~ → `github.com/stronxyo/quarter`
2. Should the first-run wizard appear automatically, or only from a key (e.g. `ctrl+k`) so the app opens straight to the list?
3. License (MIT vs GPL core + private Pro), decided before the repo goes public. See "License and monetization" in the brief.
