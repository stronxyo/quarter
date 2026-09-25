# Batch 6 Spec: Your wider library and achievements, with no key

**Branch:** `batch/06-webapi` (name kept; the batch became keyless)
**Status:** Done (approved 2026-09-25). On Pablo's PC: 331 candidates; a sample resolved to Team Fortress 2, Portal 2, Counter-Strike 2, FlatOut 2, Mafia and others. The background fill pauses for the run after a network failure.
**Goal:** zero setup for the user. Quarter shows achievement progress and the games you own but haven't installed, using only what Steam already keeps on your PC plus the public store. No Steam Web API key.

**Why keyless (Pablo, 2026-09-25):** "I want zero friction for the user." The API key plan (full account library, server playtime) moves to Rules Appendix A as an optional extra for later.

---

## What we found on Pablo's PC (2026-09-25)

- **`userdata/<accountid>/config/librarycache/<appid>.json`**: the Steam client's cached library page, one per game. A JSON list of `[section, body]` pairs:
  - `achievements`: `data.nTotal`, `data.nAchieved`, and per achievement `strName`, `strDescription`, `bAchieved`, `rtUnlocked`, `flCurrentProgress`/`flMaxProgress` (lists `vecHighlight`, `vecUnachieved`, `vecAchievedHidden`)
  - `associations`: developers and publishers; `descriptions`: full and short description
  - also `badge`, `usernews`, `gameactivity` (unused)
- Counts: 144 cached pages, **66 with achievements**; **all 11 installed games** have one.
- **Games beyond the installed ones:** 344 apps appear in `localconfig.vdf` and the library cache together; 331 are not installed. Many are DLC, soundtracks or tools; the store's `type` (`game`/`demo`) tells real games apart (Batch 5 already reads it).
- **Limit:** this is "games played or opened in Steam on this PC", not guaranteed to be the whole account library.
- `appcache/stats/UserGameStats*.bin` also exist (binary); not needed, since the JSON cache has the counts.

---

## Phase A: Achievements

**Design**
- `internal/steam/librarycache.go`: reads one game's cached page and returns achievement totals (`nTotal`, `nAchieved`). Fails soft: a missing or unreadable file means "no achievement info".
- Read for all games during the scan (a small JSON file each, local).
- Pane (Pablo's pick from previews): count first, in the value column, then a bar of middle-third blocks for unlocked (orange) and a thin line for the rest (dim amber): `Achievements 72/88   🬋🬋🬋🬋🬋🬋🬋🬋🬋🬋🬋───`. Nothing shown for games without achievements.
- Test fixtures: invented data in the real file shape.

- **Found while building (2026-09-25):** the local cache can say 0 for a game that has achievements (Dreamcore: 0 locally, 33 on the store), so the store's achievement total is fetched too (`filters=...,achievements`, one more field in the same request). Local progress wins; with only the store total the pane shows `0/33    ──────────────` (Pablo: no cached progress means zero unlocked). Elite Dangerous genuinely has no Steam achievements (the store lists none).
- Real counts on Pablo's PC: Rocket League 72/88, Watch_Dogs 2 14/55, Cyberpunk 2077 2/57, Half-Life: Alyx 2/42.

**Files:** `internal/steam/librarycache.go` (+ test), `internal/library/game.go`, `scan.go`, `internal/ui/detail.go` (+ test), `testdata/steam/userdata/2/config/librarycache/*.json`, `internal/webapi/store.go` (achievement total), `internal/config/constants.go` (cache versions)

**Done when**
- Tests: totals read; missing file and malformed file handled; bar and count drawn; games without achievements show nothing.
- By hand: your games show the same counts as the Steam client.

---

## Phase B: Games you own but haven't installed

**Design**
- Candidates: every app in `localconfig.vdf` or the library cache that isn't installed and isn't on the non-game denylist.
- Each candidate is resolved through the store (name and type), reusing Batch 5's store client, rate limit and cache. Installed games' details come first, then candidates in the background, about 1.5 s each; about 8 minutes once for Pablo's 331, then cached for a week. Games join the list as they resolve.
- Only `type` `game` or `demo` join the list; everything else is remembered as "not a game" and never asked again for a week.
- Not-installed games show an empty marker `○`; `Enter` opens Steam's install dialog (Batch 3 already does this).
- New filter: `[All]  Installed  Recent  Favourites`.
- Playtime and last played for them come from `localconfig.vdf`, as for installed games.

**Files:** `internal/library/merge.go`, `sort.go` (Installed filter) (+ tests), `internal/steam/localconfig.go` (candidate list), `internal/ui/*` (resolution, `○` marker), `cmd/quarter/main.go`

**Done when**
- Tests: candidates exclude installed and denylisted apps; only games and demos join; `○` marker; Installed filter; resolution survives a restart via the cache.
- By hand: games you've played but uninstalled appear over the first minutes, with `○`; `Enter` on one opens Steam's install window.

---

## Markers (Pablo's picks during Batch 6, from colored previews)

| Game | Marker |
|---|---|
| Installed | `▸` |
| Not installed | `▹` (white) |
| Favourite | `✦` (orange) |
| Favourite, not installed | `✧` (orange) |
| Running | `▶` (green, whole row green) |

- Small triangles chosen over large ones: they sit almost exactly centered in Cascadia Mono (0.5% off vs 2.5%).
- Checked with the font file: Cascadia Mono has no star characters, so `✦`/`✧` are borrowed from another font and can look slightly off-center. `◆`/`◇` (perfectly centered, in the font) were offered; Pablo kept the stars.

## Deferred (recorded in Rules Appendix A)

- **Steam Web API key (optional extra):** the complete account library and server playtime. Trigger: users ask for games never touched on their PC.
- **Reading `appinfo.vdf`:** names and types of every app instantly, instead of 8 minutes of store lookups on the first run. Trigger: the first-run wait bothers users, or the store limits us.

## Commits

One per phase, blocks given at the end of each phase after checking `git status`.

## PR to main

Claude writes the title and description after Phase B. Pablo opens and merges.

## Open questions for Pablo

None.
