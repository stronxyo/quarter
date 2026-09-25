# Batch 7 Spec: Game logos as text art

**Branch:** `batch/07-art`
**Status:** Approved 2026-09-25. Phase A done: a 640x360 logo draws in 0.6 ms; 259 of the 265 local logos on Pablo's PC are usable (the other 6 fall back to the CDN, then the pixel font). Phase B done: real CDN download checked with Portal (app 400); a missing logo (app 1) answered in 0.35 s and was remembered.
**Goal:** the detail pane shows the selected game's logo, redrawn in eighth-block characters with the QUARTER gradient (design G, picked in Batch 4). Every game in the list gets one, installed or not.

---

## Big libraries (Pablo's question, 2026-09-25)

Pablo has about 300 games. Will logos slow anything down?

No, because logos are never prepared ahead of time. Quarter draws only the logo of the game you have selected, when you select it:

- **Startup:** nothing changes. No logo is read while Quarter starts, so 300 or 5,000 games start just as fast.
- **Measured on Pablo's PC (2026-09-25):** all 265 local logos read and drawn one after another took 1.4 s in total. That is about 5 ms per logo: decoding takes 2 ms (95% of logos under 4.7 ms) and drawing 3 ms. The slowest logo took 85 ms. Moving through the list stays smooth.
- **Memory:** a drawn logo is about 3 KB of text. Logos already drawn this session are kept, so going back to a game is instant.
- **Not-installed games get logos too**, so the list doesn't feel half done. Steam's local cache has logos for 265 of the 656 apps it knows on Pablo's PC. The rest are downloaded from Steam's CDN, one at a time, only for the game you select, and kept on disk after that.

---

## Done before Phase A: softer text (Pablo's pick, 2026-09-25)

Plain text (list rows, pane values, the search box) is ivory `#F0E4D2` instead of the terminal's white. It was chosen from 7 previews after looking at cliamp's cream text. Secondary text stays dim amber `#B07A45`.

---

## Phase A: Drawing logos

**Design**
- New package `internal/art`: it turns an image into text art. It has no UI or Steam code, so it can be tested on its own.
- **Sampling:** a pixel counts as "on" when the logo covers at least half of its area (the area averaging from Batch 4). The logo's transparency gives its shape, which comes out in one color.
- **Eighth blocks** (2x4 dots per character) are the default: Unicode 16 "Block Octant" U+1CD00 to U+1CDE5, plus the 26 older block characters for patterns that existed before. The table goes into the code as a fixed list, generated once from Unicode's data file, and a test checks it against known characters.
- **Sixth blocks** (2x3) are the fallback for fonts without eighth blocks. They are chosen in `config.json` with `"logo_blocks": "sixths"`. There is no settings screen yet.
- **Fit:** the logo is drawn at the width and height it's given, keeping its shape. Blank rows and columns around it are trimmed, and it is centered.
- **Unusable images:** a logo with no transparency (a solid rectangle) or one that comes out almost empty counts as "no logo" and uses the fallback (Phase C).

**Files:** `internal/art/art.go`, `octants.go` (table), `sextants.go` (+ tests and a benchmark)

**Done when**
- Tests use images built in the test code (no Valve logos in the repo): a square, a circle, a transparent border that gets trimmed, a solid image rejected, eighth and sixth blocks agreeing on the same shape.
- The benchmark draws a 640x360 logo in under 10 ms.

---

## Phase B: Where logos come from

**Design**
- Sources, in order:
  1. `<steam>/appcache/librarycache/<appid>/logo.png`
  2. `<steam>/appcache/librarycache/<appid>/<hash>/logo.png` (newer Steam clients)
  3. Quarter's own download, `<data>/logos/<appid>.png`
  4. Steam's CDN: `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/<appid>/logo.png`, saved to `<data>/logos/`
- **CDN rules:**
  - Only for the selected game, after the same short pause as store lookups (250 ms), so scrolling fast downloads nothing.
  - One download at a time, with the same timeout as the store.
  - Downloads over 2 MB are refused.
  - A missing logo (404) is remembered in `<data>/logos/missing.json` for 7 days.
  - Network errors are not remembered: Quarter tries again the next time the game is selected.
- Read-only on Steam's folders. The data folder is Quarter's own.
- **Rights (not legal advice, same rule as Batch 4):** logos come from the user's own Steam cache or Steam's public CDN and are redrawn on their machine. Downloads are a private cache in the user's data folder, never in the repo, never shared. To be checked properly before any paid release.

**Files:**
- `internal/art/sources.go` (local file, CDN download, missing list) (+ test against a local test server). Moved from `steam`/`webapi` while building: the locked folder rules put logo sources in `internal/art`.
- `internal/store/store.go` (`WriteFile`: the same safe write the JSON files use, for downloaded logos)
- `internal/config/constants.go` (CDN address, limits)
- Test PNGs are made in temporary folders by the tests, not committed and not real logos.

**Done when**
- Tests: the direct file is preferred; the hashed subfolder is found; a 404 is remembered and not asked again; a network error is not remembered; an oversized download is refused.
- By hand: an uninstalled game with no local logo shows its logo after a moment, and instantly next run.

---

## Phase C: The logo in the pane

**Design**
- The logo replaces the game's name at the top of the detail pane (side by side and in the `d` pop-up).
  - Width: the pane's inner width, 36 to 64 columns.
  - Height: up to 14 rows (as previewed), but never more than 40% of the pane's height, so the details below stay visible on short windows.
- **Fixed logo area:** for a given window size, the logo area is always the same height, with each logo centered in it. The lines below it (developer, played, achievements) don't jump up and down as you move through the list.
- **Colors:** the same top-to-bottom gradient as the QUARTER logo (`colorLogoTop` to `colorLogoBottom`), so a theme change later recolors both.
- **Loading:** reading and drawing run in the background, never on the UI thread. Until the logo arrives, the game's name shows in the same area. Local logos arrive in a few ms, so this is only visible for a first CDN download.
- **No logo anywhere (Pablo, 2026-09-25: "logos are always somewhere on the web, but if all the sources are depleted we fall back to our own creation"):** the game's name is drawn big, through the same renderer, from a small built-in pixel font (A to Z, 0 to 9, basic punctuation). It is designed by us, so there's no font license question, and it matches the logos' block style. When the name doesn't fit on two lines, the pane shows the plain bold name as it does today. A preview comes before building it.
- **`o` shows or hides logos**, like `d` does for the pane. It is remembered between runs (`"hide_logos": true` in `config.json`). With logos hidden, the pane shows the bold name as today, and no logo is read or downloaded. `o logo` is listed in the `?` help.
- Drawn logos are kept in memory by game and size. A window resize draws again, at about 5 ms.

**Files:** `internal/ui/logoart.go` (loading, cache, messages) (+ test), `detail.go`, `styles.go`, `keys.go` (`o`), `settings.go`, `internal/art/font.go` (+ test), `internal/config/settings.go` (`logo_blocks`, `hide_logos`)

**Done when**
- Tests:
  - The logo shows for the selected game.
  - The logo area keeps its height across games.
  - A late result for a game no longer selected doesn't replace the current logo.
  - The name shows while loading and when there's no logo.
  - The pixel font draws every supported character.
  - `o` hides logos and is saved; with logos hidden nothing is read.
- `TestSpeedTargets` still passes (startup under 100 ms and search keystrokes under 16 ms at 5,000 games).
- By hand, on Pablo's PC: Rocket League, Elite Dangerous, Cyberpunk 2077 and Half-Life: Alyx look like the G preview. An uninstalled game shows its logo. Scrolling fast through 300 games stays smooth.

---

## Deferred (recorded in Rules Appendix A if approved)

- **Drawing logos in the list rows or a gallery view:** only the pane gets one for now.
- **`appinfo.vdf` asset paths:** only needed if Steam moves logo files again.

## Commits

One per phase, with commit blocks given at the end of each phase after checking `git status`.

## PR to main

Claude writes the title and description after Phase C. Pablo opens and merges.

## Open questions for Pablo

None. (Settled 2026-09-25: our own pixel font as the last fallback; `o` toggles logos.)
