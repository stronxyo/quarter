# Batch 4 Spec: Sixel spike

**Branch:** `batch/04-sixel-spike`
**Status:** Done (approved 2026-09-24). Decision: no images. Game logos drawn as eighth-block text art instead.
**Time box:** half a day of work. If it isn't convincing by then, the answer is "no, half-block art".
**Goal:** answer one question with evidence: can Quarter show real cover images (sixel) next to the list in Windows Terminal without flicker, leftovers or images vanishing? The answer decides how Batch 7 draws cover art.

---

## What we already know (checked 2026-09-24)

- **Windows Terminal on this PC is 1.24.** Sixel works from 1.22 (Rules Appendix C #3).
- **No new dependency needed.** `github.com/charmbracelet/x/ansi`, already in `go.mod`, ships a sixel encoder (`ansi/sixel`).
- **Bubbletea v2 has `tea.Raw`**, which writes escape codes straight to the terminal, bypassing its cell renderer. That is the way an image can get on screen.
- **The risk:** Bubbletea redraws text by comparing cells. It knows nothing about the image, so when rows under the image change it may draw text or spaces over it. The spike measures exactly that.

---

## Phase A: Prototype

A throwaway program, `cmd/sixelspike`, that is not part of Quarter:
- Left: the real library list (reusing `library.Scan`, D.R.Y.).
- Right: a reserved box showing the selected game's cover from Steam's `librarycache` (`library_600x900.jpg`, falling back to `header.jpg`), scaled to the box and drawn as sixel through `tea.Raw`.
- The same cover drawn as half-block art underneath, for a side-by-side look.
- Detects sixel support by asking the terminal (primary device attributes) and says so on screen.

**Tests, all by hand in Windows Terminal (Pablo):**
1. Move the selection quickly up and down: does the image follow without flicker?
2. Resize the window, bigger and smaller: does the image stay put and clean?
3. Search, sort, filter: does any text get drawn over the image, or leave pieces behind?
4. Zoom in and out (Ctrl + / Ctrl -): does the image keep the right size?
5. Speed: time to encode and show one cover (logged).
6. Side by side: how much better does sixel look than half-block?

**Files:** `cmd/sixelspike/main.go` (throwaway)

**Commit (so the prototype is saved in history even if it is later removed)**
```
git add cmd/sixelspike/main.go specs/BATCH_04_SPEC.md
git commit -m "spike: prototype sixel cover art beside the list, to test whether it renders cleanly in Windows Terminal."
git push
```

---

## Phase B: Decision

- Record the results of every test above in this spec.
- Decide: **sixel as an opt-in** (with half-block as the default and the fallback), or **half-block only**.
- Update Rules Appendix A (the sixel row) and the brief's Risks section with the decision and the evidence.
- If the answer is "no": delete `cmd/sixelspike` before the PR, so only the findings reach `main`. If "yes": keep it as a reference until Batch 7 replaces it.

**Files:** `specs/BATCH_04_SPEC.md`, `RULES/QUARTER_RULES_V1.md`, `PROJECT_BRIEF.md`, and possibly removing `cmd/sixelspike/`

**Commit**
```
git add specs/BATCH_04_SPEC.md RULES/QUARTER_RULES_V1.md PROJECT_BRIEF.md cmd/sixelspike
git commit -m "spike: record the sixel findings and decide how Batch 7 draws cover art."
git push
```

---

## Results (2026-09-24)

### Sixel: rejected on looks, not on technical grounds

The prototype drew the selected game's cover beside the list (sixel, with a half-block toggle). Pablo tried it and decided against images of any kind: "I want to leave the terminal look as classical as possible." The technical tests (flicker, resize, text over the image) were not needed once the look was ruled out. The prototype was deleted without being committed; the spike's value is this decision and what follows.

### What replaces it: game logos as text art

Explored with Pablo through colored previews of his own games (Rocket League, Elite Dangerous, Cyberpunk 2077, Half-Life: Alyx). Each step was compared on his screen:

| Step | Technique | Pixels per character | Verdict |
|---|---|---|---|
| A | half blocks, one point sampled | 1x2 | rough |
| B | half blocks, area averaged | 1x2 | cleaner edges |
| C, D | quarter blocks, area averaged, 56 then 64 wide | 2x2 | liked D |
| E | sixth blocks, 64 wide | 2x3 | better |
| F | soft edges (shaded half blocks) | 1x2 | rejected by Pablo |
| **G** | **eighth blocks, 64 wide** | **2x4** | **chosen** |

**The chosen design (for Batch 7):**
- Each game's `logo.png` (transparent PNG) becomes a one-color silhouette in **eighth-block characters** (Unicode 16 "Block Octant", U+1CD00 to U+1CDE5, plus 26 older block characters for the patterns that existed before), **64 columns wide**, with the **same top-to-bottom gradient as the QUARTER logo** (theme colors).
- **Area averaging:** each tiny pixel is "on" when the logo covers at least half of its area, not when one sampled point is.
- **Fallback:** sixth blocks (2x3, "Symbols for Legacy Computing") for terminals whose font lacks eighth blocks, chosen in settings.
- **Character table:** built from Unicode's own `UnicodeData.txt` (16.0.0), never typed by hand. The 26 patterns covered by older characters, with cells numbered `1 2 / 3 4 / 5 6 / 7 8`: empty = space; 1 = U+1CEA8; 2 = U+1CEAB; 12 = U+1FB82; 13 = `▘`; 24 = `▝`; 1234 = `▀`; 35 = U+1FBE6; 46 = U+1FBE7; 123456 = U+1FB85; 7 = U+1CEA3; 57 = `▖`; 1357 = `▌`; 2457 = `▞`; 123457 = `▛`; 8 = U+1CEA0; 68 = `▗`; 1368 = `▚`; 2468 = `▐`; 123468 = `▜`; 78 = `▂`; 5678 = `▄`; 135678 = `▙`; 245678 = `▟`; 345678 = `▆`; all = `█`.

**Where logos come from, in order:**
1. `librarycache/<appid>/logo.png`
2. `librarycache/<appid>/<hash>/logo.png` (newer Steam clients keep assets in hashed subfolders)
3. Steam's CDN, `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/<appid>/logo.png`, downloaded once and kept in Quarter's data folder
4. Last resort: the game's name in a block font

Measured on Pablo's PC: all 12 installed games have a local logo (5 direct, 7 in a hashed subfolder); the CDN served logos for Portal 2, Half-Life and Stardew Valley. Deeper backup if these ever miss: Steam's binary `appinfo.vdf`, which lists every asset path (stays deferred).

**Rights (not legal advice):** the logo is read from the user's own Steam cache (or Steam's public CDN) and redrawn at runtime; nothing is bundled or rehosted, the same rule as cover art in the brief. To be checked properly before any paid release.

## PR to main

Claude writes the title and description after Phase B. Pablo opens and merges.

## Open questions for Pablo

None.
