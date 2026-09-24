# Batch 3 Spec: Actions

**Branch:** `batch/03-actions`
**Status:** Draft, awaiting Pablo's approval.
**Goal:** Quarter does things, not just shows them: `Enter` plays the selected game, `i` installs it, `s` opens its store page, and `f` marks it as a favourite, with a Favourites filter.

---

## Scope decisions

- **Steam does the work.** Every action is a `steam://` link handed to the operating system, exactly as the brief says. If Steam is closed, it opens first; that's expected.
- **Opening links without a console window:** on Windows through `ShellExecute` from `golang.org/x/sys/windows` (already a dependency), not by starting `cmd` or `rundll32`, so nothing flashes on screen. `open` on macOS, `xdg-open` on Linux.
- **The link opener is passed into the UI**, so tests check which link a key produces without launching anything.
- **`Enter` on a game that isn't installed** opens the install dialog instead. Today every listed game is installed; this matters from Batch 6, when uninstalled games appear.
- **`r` (rescan)** from the brief is already done (Batch 2).
- **Package rename, `internal/cache` to `internal/store`:** favourites are saved settings, not a throwaway cache, but they need the same safe JSON save. One package serves both (D.R.Y.), so it gets a name that fits both.

---

## Phase A: Play, install, store page

**To verify first** (added to Rules Appendix C with sources): the `steam://rungameid/<appid>`, `steam://install/<appid>` and `steam://store/<appid>` link formats, and `ShellExecute` usage from `x/sys/windows`.

**Design**
- `internal/steam/launch.go`: `PlayURI`, `InstallURI`, `StoreURI` build the links; `OpenURI(uri)` hands one to the OS (`launch_windows.go`, `launch_other.go`).
- Keys (not while typing a search): `Enter` play (or install if not installed), `i` install, `s` store page.
- Feedback: a short message in the footer for 3 seconds, e.g. `▶ Launching Rocket League`, or the error if the link could not be opened.
- The open happens in a background command, never inside `Update` (Rules 4.3).
- Short help (one line) shows the most used keys; `?` lists every key.

**Files:** `internal/steam/launch.go`, `launch_windows.go`, `launch_other.go`, `launch_test.go`, `internal/ui/app.go`, `keys.go`, `app_test.go`, `cmd/quarter/main.go`, `RULES/QUARTER_RULES_V1.md`

**Done when**
- Tests: each key produces the right link for the selected game; keys typed into the search box do nothing; a failed open shows the error.
- By hand: `Enter` on a game starts it through Steam; `s` opens its store page.

**Commit**
```
git add internal/steam/launch.go internal/steam/launch_windows.go internal/steam/launch_other.go internal/steam/launch_test.go internal/ui cmd/quarter/main.go RULES/QUARTER_RULES_V1.md
git commit -m "actions: play, install and open the store page with Enter, i and s, handing steam:// links to the OS so Steam does the work."
git push
```

---

## Phase B: Favourites and a remembered view

**Design**
- Rename `internal/cache` to `internal/store` (same `Load`/`Save`, same tests).
- Settings live in `config.json` in the data folder: favourites, plus the last sort column, sort direction and filter (added by Pablo during Batch 3), so Quarter opens exactly as it was left: `{"version":1,"data":{"favourites":[620,400],"sort":"last_played","sort_desc":true,"filter":"all"}}`. Sort and filter are saved by name, not number, so reordering them in code never scrambles saved settings; an unknown name falls back to the default.
- If `config.json` exists but can't be read, it is renamed to `config.json.corrupt` (so nothing is lost) and Quarter starts with no favourites, logging a warning.
- `f` toggles the selected game; saved in the background right away.
- A `★` sits between the installed marker and the name:
  ```
    ● ★ Rocket League                                2710.5h  today
    ●   Elite Dangerous                               335.6h  6 months ago
  ```
- New filter: `[All]  Recent  Favourites`, cycled with `Tab` as today.

**Files:** `internal/store/` (renamed from `internal/cache/`), `internal/config/settings.go` (+ test), `internal/library/sort.go` (Favourites filter), `internal/ui/*`, `cmd/quarter/main.go`

**Done when**
- Tests: toggling adds and removes; favourites, sort, direction and filter survive a save and reload; an unknown saved sort or filter falls back to the default; a corrupt `config.json` is moved aside, not overwritten; the Favourites filter shows only favourites.
- By hand: star a game, sort by last played, quit, reopen: still starred, still sorted the same way.

**Commit**
```
git add internal/store internal/cache internal/config internal/library internal/ui cmd/quarter/main.go specs/BATCH_03_SPEC.md
git commit -m "settings: star favourites with f and remember them, the sort and the filter in config.json, so Quarter reopens exactly as it was left."
git push
```

---

## Small change during Phase A

- Installed marker `●` became the smaller `•` (Pablo's request).

## PR to main

Claude writes the title and description after Phase B. Pablo opens and merges.

## Open questions for Pablo

None.
