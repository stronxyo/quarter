# Logo work: handoff for the next session

> **Superseded 2026-09-28.** Logos are now real pictures (Sixel) where the terminal draws them, with one-colour text blocks as the fallback. The text-block research below is history; the decision, wins and caveats are in `specs/BATCH_08_SPEC.md`, "Decision: pictures". Its working rules (approved results are tests, check every image, keep reports short) still apply.

Read this before touching logos. It records what Pablo approved, what was tried and failed, and the mistakes that must not repeat. Written 2026-09-27, end of a long session (Batch 8, branch `batch/08-release`).

## 1. The goal, in Pablo's words

Game logos drawn in the terminal (66 x 13 cells, beside the `quarter_` box) must look like polished, well crafted pixel art: recognizable, readable words, any font, any logo. "This will end in the market."

Pablo's style picks so far:
- **One colour only**: the app's accent gradient. He saw full-colour pixel art and said "nice, but can we do it with just one colour (our accent)".
- **Pixelated is fine** if it is recognizable and clean.
- **Golf It, version "3b" then "6 dots tall, 2 dots apart"**: this is the approved target. See section 3.

## 2. How to work with Pablo (the mistakes of this session)

1. **Build on what he approved. Never replace it.** He approved a hand-tuned Golf It. I then built a "universal" automatic version, and it made Golf It unreadable again. He had to say: "all the tests and focus I asked you to do, you simply throw it away, and you just did the same all over again." Rule: an approved result is a regression test. Any general method must reproduce it exactly (same image) before it is shown.
2. **Do exactly the step he asks for, no more.** "Try it with all the other logos" meant applying the approved look to them, not re-deriving the method and dropping the parts that made it work (the crop of unreadable arc text and stars, the thick banner outline).
3. **When a general rule makes an approved logo worse, stop and say so.** Don't report "safe, changes few logos" as progress.
4. **Look at every image before reporting**, and compare with the previous approved version, not only with "today".
5. **Keep reports short and plain.** No long lists of internal knobs.
6. **Standing rules** (also in memory): never commit or push (give a git block); no em dashes or emojis; check the fixed test logos; use the Write or Edit tools for any text with backslashes (Bash and Python heredocs mangled `\n` again several times this session); no project file changes without his approval.

## 3. The approved Golf It result, and exactly how it was made

Image: `Documents\Quarter\Pixel art (research)\Golf It\Golf It - smaller, sharper letters.png`, **right panel of the bottom row ("6 dots tall, 2 apart")**.

Recipe (code in `Documents\Quarter\Prototype (logo research)\`, `py/golf2.py` and `tech/hint.go`):
1. **Letters made solid.** The logo's letters are engraved (outline plus horizontal stripes). Take the dark print inside the banner (luma < 110), label each letter separately, close each one with a 7x7 closing, fill only holes under 120 px so O and G keep their counters.
2. **Two layers.**
   - Art layer: the orange clubs plus the white banner as an outline only (banner eroded by 10 px, so a ~10 px rim), no arc text, no stars. Canvas crop `(40, 95, 600, 345)` of the 640x360 logo. File `golf_layers/h_art_thick.png`.
   - Letter layer: the solid letters on the same canvas. File `golf_layers/h_letters.png`.
3. **Art drawn by the normal renderer with dropout control on** (thin lines that are at least 3 dots long are traced along their middle so they never vanish). Without it the banner's top and bottom lines disappeared.
4. **Letters drawn hinted** (like font hinting): every letter exactly 6 dots tall on one baseline; each letter placed at the quarter-dot shift where its dots are least ambiguous; widths trimmed; letters exactly 2 dots apart; a wider space between words; the row centred between the banner's inner lines with a clear dot above and below.
5. Command that made it: `tech.exe hint <out.ans> ../golf/h_art_thick.png ../golf/h_letters.png 6,2,0.45` (fill threshold 0.45).

What made it work, and what the automatic attempt lost:
- **Cropping away parts too small to read** (arc text "PERFUSE ENTERTAINMENT", stars). This makes the banner bigger, so the letters get 6 dots. Keeping them left only 5 dots, and the serif letters broke apart.
- **Thick banner outline (10 px)**, not one dot.
- Banner position and thresholds were measured by hand on this logo. Making it automatic is the open task.

## 4. The one-colour method (approved direction, not yet in the app)

Prototype `tech/onecolor.go` plus flags in `tech/shape.go` (`dropout`, `noHollow`, `forceShift`, `lastGrid`). Recipe, all values measured on the test logos:
- The shape is today's renderer's (`internal/art/shape.go`), dot for dot.
- Each dot's colour is the plain average of its pixels. Colours are grouped into hue families (neutral when chroma < 0.06 x lightness or lightness < 0.15; new family after an empty stretch of at least 20 degrees on the hue wheel). A family is split into light and dark only when Otsu's split explains at least 0.72 of the brightness variance with a gap of at least 0.30 (grunge and bevels are ramps: 0.68 to 0.71; Golf It print is 0.74 / 0.34).
- Colour patches under 16 dots merge into their surroundings.
- Where two colours meet, the darker side gives up a one-dot gap, except for shades of the same family (or dark neutrals) that are not enclosed (a bevel, not print on a plate).
- Big fills (banners, discs) become outlines using today's own fill rule, but only the fill's usual colour is removed, so words on a banner stay (Conan ENHANCED, ShareX swirl).
- Pinholes filled; dropout control on (GRIS finally shows its rings).
- Best version: column 4 of `Documents\Quarter\Pixel art (research)\One colour\Round 2 (tuned)\`. Never worse than today on the 20 test logos. 10 to 19 ms per logo in the prototype, so it needs speeding up before going in the app (today: median 7.5 ms).

## 5. Tried and rejected (don't redo)

- Brightness-based shape: erased dark logos (Cyberpunk).
- Octants: only work in Windows Terminal's bundled Cascadia. JetBrains Mono shows `?`. Default is quarter blocks.
- Typeset titles in a matched font, and the "hybrid" (erase words, set type in place): readable but not the logo. Fragile.
- Checker or dot fills per colour on letters: noisy.
- Judging dot colour pixel by pixel: worse.
- Automatic text layer for all logos (`tech/textlayer.go`): with safety checks it only helps Apex LEGENDS, and it made Golf It unreadable. Without the checks it wrecked Rocket League, CONAN, EURO TRUCK, TACTICS. **Not the way forward as built.**

## 6. Research done (4 agents, sources in the session)

- Terminal renderers (chafa, tiv, timg, notcurses): best symbol plus 2 colours per cell. Braille is weak for images.
- Downscaling: DPID (Weber et al. 2016) edge-aware; Gerstner 2012 pixel abstraction; font hinting and stem darkening; potrace (pure Go: gotranspile/gotrace).
- Tiny text: readable floor about 5 to 7 px cap height; favicon practice is to drop small detail and taglines, thicken, add spacing; no terminal renderer special-cases text.
- Colour: OKLab distance; luminance ramps; no dithering for logos.

## 7. Suggested next step (ask Pablo first)

Start from the approved Golf It and generalise it one piece at a time, re-rendering Golf It after every change and checking it still matches the approved image:
1. Automatically drop parts too small to read at band size (tiny arc text, stars, taglines, trademark marks), then re-fit the rest bigger.
2. Automatically find a plate (banner) with print on it, and draw it as a thick outline.
3. Only then apply the hinted letters to other logos, showing each one next to today and next to the original.

## 8. Where things are

- Test logos: the 12 in `specs/BATCH_08_SPEC.md` Phase A, plus the hard 8: 1579380 (Shadow Tactics), 571740 (Golf It), 678950 (Dragon Ball), 331360 (Heavy Metal Machines), 227300 (Euro Truck 2), 381210 (Dead by Daylight), 1172470 (Apex), 683320 (GRIS). Logo files: `C:\Program Files (x86)\Steam\appcache\librarycache\<appid>\logo.png`.
- Prototype code and tools: `Documents\Quarter\Prototype (logo research)\` (`tech` Go prototypes with `run.sh`, `render` turns .ans into PNG, `py` Golf It layer scripts, `golf_layers` their outputs). Paths inside the scripts point at the old temporary scratch folder and need updating.
- All comparison sheets: `Documents\Quarter\Pixel art (research)\`.
- App code: `internal/art/` (renderer), `internal/ui/logoart.go` (band). Nothing from this research is in the app yet.
