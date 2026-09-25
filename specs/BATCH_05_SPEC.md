# Batch 5 Spec: Detail pane

**Branch:** `batch/05-detail`
**Status:** Done (approved 2026-09-24). Added during the batch by Pablo: the pane grows to 68 columns, the running game in green, and the accent moved to the middle of the logo gradient (#FFAF63).
**Goal:** the selected game's details beside the list: its name (the logo slot, filled in Batch 7), developer, release year, genres, playtime, last played, size and a short description from the Steam store.

---

## Layout (Pablo's pick: side by side)

```
  ▸─ [All]  Recent  ── 11 games ────   ╔═══════════════════════════════════╗
      Name              Played  Last   ║ ROCKET LEAGUE                     ║
│ • Rocket League      2710.5h  today  ║ Psyonix LLC · 2015                ║
  • Elite Dangerous     335.6h  6 mo   ║ Action, Indie, Racing, Sports     ║
  ✦ Dreamcore             3.2h  2 mo   ║                                   ║
  • ShareX               54.9h  6 wk   ║ Played 2710.5h · last today       ║
                                       ║ 12.4 GB on C:                     ║
                                       ║                                   ║
                                       ║ Rocket League is a high-powered   ║
                                       ║ hybrid of arcade-style soccer...  ║
                                       ╚═══════════════════════════════════╝
```

- The pane sits right of the list in the same double-line gold box as the QUARTER logo. The big QUARTER logo stays on top, across both.
- **The pane grows with the window** (Pablo's call during Phase A): the table takes up to 80 columns first, then the pane grows from 40 to 68 columns, so from about 150 columns wide the game logo (Batch 7) gets its full 64 columns.
- The list gives up width to the pane: the table shrinks from 80 toward a minimum of 56 columns; names are cut with `…` when needed.
- **Narrow windows** (too small for list + pane): the pane is hidden and `d` opens it as a pop-up over the list; `esc` closes it. On wide windows `d` does nothing new.
- The logo slot shows the game's name in bold gold until Batch 7 draws the real logo there.

## What we checked (2026-09-24)

- `store.steampowered.com/api/appdetails?appids=<id>&l=english&filters=basic,developers,publishers,genres,release_date` returns everything needed in ~12 KB (35 KB without `filters`). Checked on Rocket League and Cyberpunk 2077.
- **The answer can be keyed by a different ID** than the one asked for (Rocket League 252950 came back under `1351431`, Cyberpunk 1091500 under `2441600`). The code reads the single entry returned and checks `data.steam_appid` matches.
- `type` is `"game"` or `"demo"` (Slow Roads Demo: `demo`, with `fullgame` pointing at the full game), so the store's type replaces the name guess for `(Demo)` when known.
- Rate limit: about 200 requests per 5 minutes (Rules Appendix C #1).

---

## Phase A: The pane, with local data

**Design**
- `internal/ui/detail.go`: renders the pane from what Quarter already knows: name, played, last played (relative, like the list), size on disk, which drive/library, favourite, `(Demo)` tag.
- Side by side when the window is wide enough; otherwise `d` opens it as a pop-up, `esc` closes.
- The pane follows the selection.

**Files:** `internal/ui/detail.go`, `detail_test.go`, `app.go`, `keys.go`, `styles.go`, `list.go` (table width now flexible), `internal/library/format.go` (size formatting)

**Done when**
- Tests: pane shows the selected game's data; wide windows show it beside the list; narrow windows hide it and `d`/`esc` open and close it; nothing overflows the window.
- By hand: move through your games and watch the pane follow.

**Commit**
```
git add internal/ui internal/library/format.go internal/library/format_test.go specs/BATCH_05_SPEC.md
git commit -m "detail: show the selected game's playtime, last played, size and library beside the list, as a pop-up on narrow windows."
git push
```

---

## Phase B: Store details

**Design**
- `internal/webapi/store.go`: fetches `appdetails` for one game with a 10-second timeout; parses name, type, developers, publishers, release date, genres, short description (HTML entities decoded).
- **Rate limiter:** at most one request every 1.5 seconds (200 per 5 minutes), shared by everything that calls the store.
- **When:** for the selected game, a quarter-second after the selection stops moving (so scrolling fast doesn't fire requests); then, slowly in the background, for the rest of the installed games.
- **Cache:** `store.json` in the data folder, per game, with the fetch time. Refreshed after 7 days. A failed fetch is remembered for 1 day so a missing or delisted game isn't asked about again and again.
- **Offline or failed:** the pane still shows everything local, plus a dim "store info unavailable".
- `type: demo` from the store sets the `(Demo)` tag even when the name doesn't say "Demo".

**Files:** `internal/webapi/store.go`, `store_test.go` (tested with `httptest`, never the real store), `internal/library/game.go` (store fields), `internal/ui/detail.go`, `app.go`, `internal/config/constants.go`, `cmd/quarter/main.go`, `RULES/QUARTER_RULES_V1.md` (Appendix C: the key mismatch and filters)

**Done when**
- Tests: parsing a real saved response; the key-mismatch case; a failed request; the rate limiter spacing; the cache refresh and failure windows; demos tagged from `type`.
- By hand: every one of your games shows its developer, year, genres and description.

**Commit**
```
git add internal/webapi internal/library internal/ui internal/config/constants.go cmd/quarter/main.go RULES/QUARTER_RULES_V1.md specs/BATCH_05_SPEC.md
git commit -m "detail: add developer, year, genres and description from the Steam store, rate limited and cached for a week."
git push
```

---

## Phase C: Running game (added by Pablo during Batch 5)

**Verified 2026-09-24 on Pablo's PC:** with Rocket League running, `HKCU\Software\Valve\Steam\RunningAppID` read `252950` and `HKCU\Software\Valve\Steam\Appsª950\Running` read `1`; with nothing running, `RunningAppID` is `0`.

**Design**
- `internal/steam/running.go`: `RunningApp() (appID uint32, known bool)`; Windows reads `RunningAppID` from the registry. **Linux and macOS report "unknown"**: checked during the phase, Steam for Linux stopped writing `RunningAppID` to `registry.vdf` in 2023 (issue still open). A process-list fallback is deferred (Rules Appendix A).
- The UI checks every 2 seconds in a background command.
- The running game's row turns green and its Last played column reads `▶ running`. The detail pane shows a green `▶ RUNNING NOW`.
- `Enter` on the running game shows "Rocket League is already running" instead of launching it again.

**Files:** `internal/steam/running.go`, `running_windows.go`, `running_other.go`, `internal/ui/*`, `internal/config/constants.go`, `RULES/QUARTER_RULES_V1.md` (Appendix C)

**Done when**
- Tests: the running game is marked in the list and pane; the mark clears when the game stops; `Enter` on it doesn't relaunch.
- By hand: start a game from Quarter; within 2 seconds its row turns green.

**Commit**
```
git add internal/steam internal/ui internal/config/constants.go RULES/QUARTER_RULES_V1.md specs/BATCH_05_SPEC.md
git commit -m "running: mark the game Steam is running in green, read from Steam's RunningAppID every 2 seconds."
git push
```

---

## PR to main

Claude writes the title and description after Phase B. Pablo opens and merges.

## Open questions for Pablo

None.
