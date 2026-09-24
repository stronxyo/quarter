# Batch 2 Spec: List UI

**Branch:** `batch/02-list-ui`
**Status:** Draft, awaiting Pablo's approval.
**Goal:** the first real screen. Running `quarter` opens a full-screen library list in the terminal: move with `j/k`, search with `/`, change sort and filter, quit with `q`. It opens instantly from a cache and refreshes in the background.

---

## Scope decisions

- **List pane only.** The detail pane is Batch 5; actions (play, install, favourite) are Batch 3.
- **Filters now: All and Recent** (played in the last 14 days). "Installed" and "Favourites" filters arrive with the data that makes them meaningful (Batch 6 and Batch 3). Today every listed game is installed, so an Installed filter would do nothing.
- **Use Bubbles' `list` component** instead of writing our own list. It already has scrolling, paging and `/` fuzzy filtering (built on `sahilm/fuzzy`, our locked choice). We only write a custom row renderer. D.R.Y.: trust existing working code. **Fallback:** if it can't meet the Phase D benchmarks or the layout, a custom list replaces it, and Pablo is told first.
- **No `teatest`.** For Bubbletea v2 it only exists as an untagged experimental module. UI tests call the model's `Update` and `View` directly (plain `testing`, no dependency). Rules section 3 is updated to match.
- **`quarter` with no flags opens the UI.** `--list` and `--version` keep working.
- **Logging starts here:** the terminal is taken by the UI, so warnings go to `quarter.log` in the app data folder (`%APPDATA%\quarter\` on Windows).
- **You check the look by hand** in Windows Terminal at the end of each phase: I can run tests but can't see the screen.

**New dependencies (locked stack):** `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`.

---

## Phase A: App shell and list

**Design**
- `internal/config/paths.go`: app data folder (`%APPDATA%\quarter`, `~/.config/quarter` elsewhere), created on first use.
- `internal/ui/app.go`: root model. States: scanning, ready, error (Steam not found). The scan runs in a `tea.Cmd` and arrives as a message (Rules 4.3).
- `internal/ui/list.go`: Bubbles list with a custom row: installed marker, name, hours right-aligned.
- `internal/ui/styles.go`, `internal/ui/keys.go`: all colors and keybindings in one place (Rules 4.4).
- Keys: `j/k` and arrows move, `g/G` top/bottom, `pgup/pgdn`, `q` and `ctrl+c` quit.
- Status bar: game count, and a warning count when files failed to read (details in `quarter.log`).
- Full screen (alt screen); redraws correctly on window resize.

**Files:** `cmd/quarter/main.go`, `internal/config/paths.go`, `internal/ui/app.go`, `list.go`, `styles.go`, `keys.go` (+ `_test.go`), `go.mod`, `go.sum`

**Done when**
- Tests: scan message fills the list; `j/k` move the selection; `q` quits; Steam-not-found shows an error screen, not a crash.
- By hand: `quarter` shows your 11 games full screen; resizing the window redraws cleanly.

**Commit**
```
git add go.mod go.sum cmd/quarter/main.go internal/config/paths.go internal/ui
git commit -m "ui: open a full-screen library list when quarter runs, with the scan in the background so the screen never freezes."
git push
```

---

## Phase B: Search, sort and filter

**Design**
- `/` opens fuzzy search as you type; `esc` clears it.
- **Columns (changed by Pablo during Phase B):** Name, Played and Last played are all visible, with a column header. `n`, `p`, `l` sort by that column; the same key again reverses it. The sorted column shows an arrow. Last played reads like the Steam client ("today", "11 days ago", "6 months ago").
- `Tab` cycles filter: All, Recent.
- Header line shows the title, the active filter and the game count ("6 of 11 games" while searching).
- The selected game stays selected when sort or filter changes (if it is still visible).

**Files:** `internal/ui/list.go`, `internal/ui/app.go`, `internal/library/sort.go` (+ tests), `internal/config/constants.go` (recent window: 14 days)

**Done when**
- Tests: each sort order is correct (ties broken by name); Recent keeps only games played in the last 14 days; search narrows the list; selection survives a sort change.
- By hand: typing `/rock` jumps to Rocket League.

**Commit**
```
git add internal/ui internal/library/sort.go internal/library/sort_test.go internal/config/constants.go
git commit -m "ui: add fuzzy search, sort by name, playtime or last played, and a recent filter, so any game is a few keys away."
git push
```

---

## Phase B2: Logo (added by Pablo during Batch 2)

**Design**
- The big "ANSI Shadow" QUARTER logo at the top, with a top-to-bottom gradient (coin gold into orange), picked by Pablo from colored previews.
- "More air" spacing (Pablo's pick): an empty row above, a 4-column left margin, two empty rows below.
- The logo sits centered in a double-line gold box (matching the logo's shadow lines) as wide as the table (Pablo's pick), so the box's right edge lines up with the end of the section line.
- Shown as long as the list still gets at least 8 rows (or every game, when fewer than 8 are showing) and the window is wide enough; otherwise the compact `QUARTER` badge takes over. Recomputed on every update, since searching and filtering change how many games show.
- Section line under the logo (Pablo's pick, cliamp style): `▸─ [All]  Recent ── 11 games ────`, a dim rule to the table edge. While searching, the search box sits on this line (`── Search: roc ── 2 of 11 games ──`), so no extra row is needed.
- The font is the terminal's choice, not Quarter's. JetBrains Mono is recommended in the README (Batch 8).
- The gradient colors are named theme colors in `styles.go`, so future themes recolor the logo.

**Files:** `internal/ui/logo.go`, `internal/ui/styles.go`, `internal/ui/app.go`, `internal/ui/app_test.go`

**Done when:** tests show the logo when tall, the badge when short or narrow, and the view never taller than the window.

---

## Phase C: Instant start from cache

**Design**
- `internal/cache`: load and save JSON files in the app data folder. Writes go to a temp file then rename, so a crash never leaves a half-written cache.
- On launch: load `library.json` and show it immediately, then rescan in the background and swap in the fresh list (keeping the selection).
- The cache has a format version. A missing, corrupt or old-version cache is ignored and rebuilt; it never blocks startup.
- `r` forces a rescan (the refresh key from the brief).

**Files:** `internal/cache/cache.go` (+ test), `internal/ui/app.go`, `cmd/quarter/main.go`

**Done when**
- Tests: save then load round-trips; corrupt file and wrong version return "no cache" without error to the user; interrupted write leaves the old cache intact.
- By hand: second launch shows the list immediately, no "Scanning..." flash.

**Commit**
```
git add internal/cache internal/ui/app.go cmd/quarter/main.go
git commit -m "cache: show the library from a local cache on launch and rescan in the background, so Quarter opens instantly."
git push
```

---

## Phase D: Benchmarks

The brief's targets, made measurable:
- **Startup:** load a 5,000-game cache, build the model, render the first frame: **under 100 ms**. Terminal start-up itself is outside our control and not counted.
- **Search:** filter 5,000 games for one keystroke and render: **under 16 ms** (one frame at 60 fps), which is what "no dropped keystrokes" means in practice.

**Design**
- A generated 5,000-game fixture (made by the benchmark, not stored in the repo).
- `go test -bench` benchmarks for both targets, plus a test that fails if either target is missed, so a regression breaks the build.

**Files:** `internal/ui/bench_test.go`, `RULES/QUARTER_RULES_V1.md` (teatest removal, stack table), `specs/BATCH_02_SPEC.md`

**Done when:** both targets pass on this PC, with the numbers recorded in this spec.

**Commit**
```
git add internal/ui/bench_test.go RULES/QUARTER_RULES_V1.md specs/BATCH_02_SPEC.md
git commit -m "perf: add startup and search benchmarks for a 5,000-game library, so the speed targets are checked on every run."
git push
```

---

## PR to main

Claude writes the title and description after Phase D. Pablo opens and merges.

## Open questions for Pablo

None.
