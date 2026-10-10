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
- **Phase A, toolchain:** install Go (`winget install GoLang.Go`) and staticcheck 2026.2.1. Third-party libraries are added in the batch that first imports them.
- **Phase B, skeleton:** `go mod init github.com/stronxyo/quarter`, `cmd/quarter/main.go` with `--version`, `internal/config/constants.go`.
- **Phase C, verify claims:** research every item in Rules Appendix C, with Sources. Update the brief and rules with the results.
- Done when: `go run ./cmd/quarter --version` prints a version; every Appendix C item is marked verified or wrong, with a source.

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
- Outcome: rejected then (keep the classic look); reversed in Batch 8, when logos became Sixel pictures at Pablo's request.

### Batch 5: Detail pane (`batch/05-detail`)
- **Phase A, layout:** two panes; collapse to overlay on narrow terminals.
- **Phase B, store metadata:** `appdetails` for the selected game, rate limiter, cache (refresh window: days).
- Done when: selecting a game shows developer, year, genres, description, playtime.

### Batch 6: Web API (`batch/06-webapi`)
- **Phase A, setup:** API key wizard (SteamID64 auto-detected), plain-text storage notice.
- **Phase B, owned games:** `GetOwnedGames` merges uninstalled games; private-profile detection and message.
- **Phase C, achievements:** fetched lazily for the selected game, progress bar.
- Done when: uninstalled games appear with `○`; achievements show in the detail pane.

### Batch 7: Game logos (`batch/07-art`)
- **Phase A, render:** eighth-block text art, 64 wide, area averaged; sixth blocks as a setting for fonts without eighth blocks. Design in `specs/BATCH_04_SPEC.md`.
- **Phase B, sources:** `logo.png` from Steam's `librarycache/<appid>/`, then its hashed subfolder, then Steam's CDN (kept in the data folder).
- **Phase C, pane:** the logo in the QUARTER gradient at the top of the detail pane, `o` to hide logos, and our own pixel font when no logo exists anywhere.
- Done when: the selected game's logo shows in the detail pane, for every game in the list. Spec: `specs/BATCH_07_SPEC.md`.

### Batch 8: Polish and release (`batch/08-release`)
- **Done first, logos:** real pictures (Sixel) in amber tones where the terminal draws them, one-colour text blocks elsewhere. Spec: `specs/BATCH_08_SPEC.md`.
- **Phase A, UX:** `?` help overlay, "INSERT QUARTER" start screen, a random goodbye line on quit; a setting to turn logo pictures off.
- **Phase B, error states:** Steam not found, no libraries, offline.
- **Phase C, performance:** re-check the brief's targets; `pprof` any misses; logo pictures with very fast scrolling on a large library, and inside tmux.
- **Phase D, release:** GoReleaser (Windows/macOS/Linux, amd64 + arm64, stripped binaries); smoke test on a clean terminal and, if available, a Steam Deck.
- Done when: v0.1 tagged.

### Batch 9: Pictures on Kitty and Ghostty (`batch/09-kitty-pictures`)
- Kitty and Ghostty don't draw Sixel; they have their own image protocol (the Kitty graphics protocol). Today they get the text fallback.
- **Phase A, research:** how each terminal announces the protocol, how a picture is placed in a cell area and removed, how it behaves with redraws; verified with sources, per the rules.
- **Phase B, render:** the same pictures as Sixel (same crop, fit, amber tones and accent rule, from `internal/art`), sent through the Kitty protocol instead; Sixel stays for the terminals that have it.
- Done when: the 20 test logos look the same in Kitty and Ghostty as in Windows Terminal, checked side by side.

## Session history (moved into Batch 8)

Pablo's idea (2026-09-29): a history of the games actually played (runs only, never what was scrolled past), arranged by most played and split into periods. This is the "stats" feature in the brief. Planned for after v0.1; moved into Batch 8 the same day, before Phase C, once Steam's own log gave six months of real sessions to design against. Built: the ranking and the HIGH SCORES board on `h` (`specs/BATCH_08_SPEC.md`, "Session history"). Then the past years from Steam (Steam Replay, 2022 on, opt-in from history). The ideas below that aren't built stay for later.

- **Why it needs its own record:** Steam keeps only each game's total hours and last played date (`localconfig.vdf`), no history. Quarter has to build it and cannot recover the past, so recording could start earlier than the screen (open question: start recording in Batch 8).
- **Source 1, sessions Quarter sees:** it already polls the running game every 2 s. Each run saved with game, start and length: exact sessions.
- **Source 2, play while Quarter was closed:** at each start, compare every game's total hours with the last saved. The difference is time played in between, with an approximate date (between the previous check and the last played date). Keeps the history complete when games are launched from Steam directly.
- **Periods, by most played:** last 7 days, 1 month, 3 months, 6 months, 1 year, all time (all time from Steam's exact totals). Hours and an amber bar per game.
- **Calendar of played days:** a year of small squares like GitHub's contribution grid, the day's shade showing how much was played; half blocks in the amber palette.
- **In the detail pane:** "this month: 12h, last session: yesterday, 1h 20m".
- **Year recap ("Quarter Wrapped"):** top game, total hours, longest session, most played day of the week.
- **Longest streak** of days in a row playing.
- **Rediscover:** games played a lot but not opened for 6 months or more.
- **Local only:** one small file in Quarter's data folder; nothing leaves the PC (goes in the privacy note).
- **Where:** its own screen on one key (e.g. `h`), left and right moving between periods like the pages.
- **Suggested order:** recording and the detail-pane line first (small, safe, starts the data); the full screen, calendar and recap once there are weeks of real data to design against.

## After v0.1: more launchers

The v1.0 goal is one terminal library for every PC launcher. One launcher per batch, in this order unless Pablo changes it: Epic Games, GOG, Ubisoft Connect, Battle.net, Xbox / Microsoft Store (PC Game Pass), EA app, then Amazon Games, itch.io and others as asked. Each batch has the same phases:
- **Phase A, research:** where the launcher keeps its installed games, how a game is started, whether owned but uninstalled games can be listed, where logos come from. Nothing is assumed: every claim verified with sources, per the rules.
- **Phase B, scan:** installed games read into the same `Game` model, marked with their launcher; parsers fail soft and are tested against `testdata/`.
- **Phase C, launch and logos:** play and install through the launcher; logos through the same picture and text renderers.
- **Phase D, UI:** a launcher filter, and the launcher shown in the detail pane.

The v0.2 and v0.3 features in the brief (collections, backlog, stats) are planned alongside; the order is decided before v0.1 ships.

## Before v0.1: Steam starts quietly

Pablo moved this before the release on 2026-10-07: when Quarter has to start Steam to play a game, Steam starts in the tray without its window. Its own session; everything is in `specs/STEAM_QUIET_START.md`.

## After v0.1: games from outside the launchers

Pablo asked on 2026-10-07 for games on the PC that didn't come through Steam, with as little friction for the user as possible. Not started; ideas in order of least friction, each to be verified in its own research phase:
1. **Steam's non-Steam games:** games the user already added to Steam ("Add a Non-Steam Game") are listed in Steam's `userdata/<id>/config/shortcuts.vdf` (a binary VDF) on every system, and can be started through Steam. Nothing to ask the user. To verify: the file format, the id used to launch them, and where their artwork lives.
2. **The other launchers:** Epic, GOG, Ubisoft, Battle.net, Xbox, EA (the section above). Found by reading each launcher's own records, nothing to ask.
3. **A folder the user adds:** a games folder (for example `D:\Games`) picked once in the settings; Quarter lists the games in it. The hard part is choosing the right program inside each game's folder among launchers, uninstallers and crash reporters: guess from the folder and file names and sizes, and let the user correct it once. Never search the whole PC.
4. **Windows' installed programs list** (the Uninstall keys in the registry): finds anything installed with an installer, but mixes games with every other program; Windows-only, so at most a helper for 3.
- **Rules:** nothing scanned without the user knowing; no folder outside what the user picked; logos where a source exists, the text fallback otherwise.

## Testing

- Unit tests for `vdf`, `steam/*` parsers, `filter`, `library/merge`, run against `testdata/`
- `webapi` tested with `httptest` servers, never the real API in tests
- UI: key flows tested by calling the model's `Update` and `View` directly (no `teatest`); the look checked by hand in Windows Terminal
- Speed: `go test ./internal/ui -bench .` for numbers; on 5,000 games and 20,000 sessions, `TestSpeedTargets` fails the build if startup (saved files to the first frame) passes 100 ms, or a search keystroke, a held step with pictures, a history key or a download's blink passes 16 ms; `TestIntroFrameSpeed` if an intro frame passes 33 ms (Phase C, `specs/BATCH_08_SPEC.md`)

## Open questions

1. ~~Go module path~~ → `github.com/stronxyo/quarter`
2. Should the first-run wizard appear automatically, or only from a key (e.g. `ctrl+k`) so the app opens straight to the list?
3. License (MIT vs GPL core + private Pro), decided before the repo goes public. See "License and monetization" in the brief.
