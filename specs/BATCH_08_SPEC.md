# Batch 8 Spec: Polish and release

**Branch:** `batch/08-release`
**Status:** Phase A (logo drawing) done 2026-09-28: logos are real pictures (Sixel) where the terminal draws them, one-colour text blocks elsewhere. See "Decision: pictures" below. The other phases follow `PLAN.md` and get their own spec sections before code.

---

## Phase A: Logo drawing, rebuilt from a written target

Batch 7 tuned the logo drawing one logo at a time, and fixes for one logo kept breaking others. This phase restarts it from a target Pablo set, checked on a fixed test set every time.

### The target (Pablo, 2026-09-26)

1. **Every font:** only characters every common monospace font has (block elements U+2580 to U+259F). JetBrains Mono showed `?` for the octants.
2. **Fill spaces fixed:** big filled areas (banners, plates, discs) become a clean outline one dot wide, not a painted block.
3. **Thick and thin letters well seen:** letters stay solid; thick ones keep their weight, thin ones stay visible, none merge.
4. **Retro look, each word formed:** crisp pixels; each letter readable, not a blob.

### The test set (12 of Pablo's logos)

Cyberpunk 2077 (dark logo, thin strokes), THE FINALS (heavy letters, thin gaps), Rocket League, Portal 2, Elite Dangerous (emblem, small print), Half-Life: Alyx (hair-thin letters), APB Reloaded (letters cut into a plate), Conan Exiles (banner), ShareX (disc), Z1 Battle Royale (dark rust-red letter), Worms Revolution (heavy joined letters), Borderlands 2 (joined letters).

A change ships only if it is at least as good as the last approved version on all 12. The approved sheet is kept in Pablo's Documents (`Quarter/Logo test set.png`).

### How it works (`internal/art/shape.go`)

- **Shape from transparency.** In a mostly bright logo, near-black pixels (brightness under 28 of 255) are outlines and count as gaps. A dark coloured letter (Z1's rust-red Z) is still a letter. A mostly near-black logo (Cyberpunk, 74% near black) is drawn by its transparency.
- **Specks:** dark patches smaller than a dot are texture and are filled, so grunge doesn't shatter letters.
- **Fills:** a connected piece that is at least 20% of the logo and averages at least 12 dots thick is hollowed to a rim. Measured on the test set: the fills (Conan's banner, ShareX's disc) average 12.6 and 13.3 dots; the heaviest lettering (Worms) 11.2.
- **Thin and thick, judged locally:** strokes thinner than a dot count three times over, so lines don't vanish; gaps and holes narrower than a dot keep their dots off, so letters stay apart and counters stay open.
- **Grid fitting:** the dot grid is shifted by quarter dots to where edges land on whole dots.
- **Octants** stay available (`"logo_blocks": "octants"` in `config.json`) for fonts that have them: twice the detail for small print.

### Limits, recorded honestly

- Small print two rows tall (Conan's EXILES, Alyx's HALF-LIFE) can't be read in quarter blocks; octants handle it.
- Drawing time over Pablo's 265 logos: median 7.5 ms, slowest 39 ms, off the UI thread and cached.

### Study: the hardest logos and six techniques (2026-09-27)

Pablo: "we are not there yet", show the most difficult logos with different techniques. Done in a scratch copy of the renderer; nothing in the project changed.

**Choosing the hard logos by measurement.** All 273 logos on Pablo's PC (Steam's cache plus Quarter's downloads) were scored at the band size (66 x 13 cells) on stroke thickness, share of small print, dark share, number of pieces and texture specks. The 12 hardest, varied, plus Cyberpunk 2077 and Conan Exiles as references:

| Logo | What makes it hard |
|---|---|
| DRAGON BALL FighterZ | dark red letters painted on a bright ball; 1,107 texture specks |
| GRIS | hair-thin (0.3 dots), half-transparent lines; the title is under one row tall |
| Golf It! | dark text on a white banner; 101 pieces |
| Heavy Metal Machines | thin, dark-outlined metal letters |
| The Pedestrian | text inside a plate |
| Jurassic World Evolution | emblem with small print |
| Shadow Tactics: Aiko's Choice | small print |
| Euro Truck Simulator 2 | heavy italic letters over a disc |
| Among Trees | letters under one row tall, widely spaced |
| Dead by Daylight | several sizes of text |
| Apex Legends | heavy letters with cracks |
| Little Nightmares | tall thin letters |

**The six techniques** (sheet kept in Pablo's Documents: `Quarter/Hard logos - 6 techniques.png`, originals in `Quarter/Hard logos - originals.png`):

| | Technique | Characters | Result |
|---|---|---|---|
| A | Quarter blocks (current default) | every font | fine on bold logos; loses fine print |
| B | Braille dots | every font (#22) | octant detail as dots; an "LED sign" look |
| C | Octants | Windows Terminal's own Cascadia only (#22) | sharpest: Conan's EXILES and Shadow Tactics' small print read |
| D | Blocks plus Braille for fine detail (chafa's idea) | every font | the two textures clash; dropped |
| E | The game's name in the retro pixel font | every font | always readable; the only answer for GRIS and Among Trees |
| F | Two-tone: bright parts in the gradient, dark parts in dim amber (a cell's glyph colour plus its background colour) | every font | reveals what one colour can't: Conan's banner says ENHANCED, Dead by Daylight's NEW CHAPTER AVAILABLE, depth on Apex and Little Nightmares. Fails on logos whose main body is dark (Cyberpunk, DRAGON BALL, Jurassic World): they go brown. Untested fix: give the gradient to the tone that covers more of the logo. |

**The limit.** A character cell is the smallest thing a terminal draws. A title under one row tall in the original (GRIS, Among Trees) cannot become a readable word with any characters or any tool. No skill exists for this (the registry has figlet text generators only), and chafa uses the same characters (#24), so it can't exceed this either.

This study's recommendation (two-tone art, the name in the retro font, octants as a setting) was overtaken by the decision below.

### Better text blocks: tried and rejected (2026-09-27 to 09-28)

Several days of work to make the hard logos readable in text blocks. Everything is kept in Pablo's Documents (`Quarter/Logo lab/`); nothing of it is in the app.

- **Round 1, four agents on the 8 hardest logos:** layered rebuild (drop small parts, banners as outlines, hinted letters), vector redraw, optimisation search, and a hand-drawn target. Only the hand-drawn one beat today; the automatic ones broke as many logos as they fixed.
- **Round 2, two automatic templates** built from the hand-drawn decisions (mask search, colour-based text detection). Clean on Dead by Daylight, Apex, Euro Truck; failed on brush and metal lettering (Shadow Tactics, DRAGON BALL, Heavy Metal) and lost Golf It's banner.
- **Amber tones in text blocks** ("crisp: 4 amber tones", main colour as the accent): added real information (Golf It's letters, Portal 2's figure), but its own shape melted letters together (THE FINALS, Rocket League), and every clean-up pass added noise. Putting the tones on the one-colour shape kept the edges but didn't read better.
- **The lesson:** a cell of 2 x 2 dots was the limit behind every failure. Improving one logo broke another. Pablo: "any time you try to improve you break and add some ugly stuff". What removed the limit was drawing real pixels.

### Decision: pictures (Pablo, 2026-09-28)

Pablo compared real pictures in amber tones (smooth, chunky x3, chunky x5) with today's text blocks and picked the smooth one ("A. real image, amber tones"). Seen in his Windows Terminal: "the massive extremely perfect point".

**How it works.**
- `internal/art/sixel.go`, `RenderImage`: the logo cropped to its visible part, fitted and centred in the band in pixels (an area average when shrinking), transparent around it.
- **Colour rule** (`internal/art/colour.go`, `amberTones`): the logo's colours are grouped into up to five in OKLab. The group covering the most of the logo becomes the app accent (`#FFAF63`), the brightest tone. The others step down from it in the accent's hue (at most 0.11 OKLab lightness per step, no darker than 0.40), nearest in the logo's own lightness first, so nothing outshines the accent. A one-colour logo is all accent.
- A game with no logo shows its name in Quarter's pixel font as a picture, in the logo gradient (`LinesImage`).
- `internal/ui/picture.go`: at start Quarter asks the terminal for Sixel support (device attribute 4) and its cell size in pixels (`CSI 16 t`). With Sixel, the band stays blank on screen and the picture is written into it directly (`tea.Raw`), 60 ms after the screen has drawn, the previous picture erased first. It is redrawn when the game, the band or the window size changes.
- Without Sixel: the one-colour text blocks, unchanged (Pablo: the cleanest of the text versions).
- Pictures are drawn off the UI thread and cached per game and band size, like the text logos. 12 to 37 KB per logo at 660 x 260 pixels.

**Wins.**
- Every hard logo reads, small print included: GOLF IT with its banner and clubs, DRAGON BALL FighterZ, HEAVY METAL MACHINES, EURO TRUCK SIMULATOR 2, AIKO'S CHOICE, NEW CHAPTER AVAILABLE, GRIS's hair-thin rings.
- One renderer for every logo: no per-logo rules, no thresholds tuned on single logos.
- The accent is the main colour of every logo, so the band matches the app.
- Checked: the app's pictures are byte for byte the ones Pablo approved, on the 20 test logos (8 hard plus the 12-logo set).

**Caveats, recorded honestly.**
- **Terminals.** Sixel is drawn by Windows Terminal 1.22 and later (tested on Pablo's PC) and, per their documentation (not tested by us), WezTerm, iTerm2, foot, Konsole and mlterm. Kitty and Ghostty don't draw Sixel; they use the Kitty graphics protocol and get the text fallback for now (planned: Batch 9).
- **Cell size.** If the terminal doesn't answer `CSI 16 t`, a cell is taken as 8 x 16 pixels, so the picture may be smaller than the band but never spills out of it.
- **Drawn outside the screen renderer.** A full redraw clears pictures; Quarter redraws them on window size changes, but any other full redraw would leave the band empty until the selection changes. Not yet tested: inside tmux (needs Sixel passthrough), and very fast scrolling through a large library. Both go into Phase C (performance).
- **No setting to turn pictures off yet.** Proposed for the UX phase, like `logo_blocks`.
- The chunky variants lost GRIS's hair-thin lines; the smooth one Pablo chose keeps them.

### Phase A mistakes, so they aren't repeated

- Judging by brightness alone erased dark logos (Cyberpunk). Judge by transparency; only near black is a gap.
- Tuning on one or two logos broke others. Always check the whole test set.
- A commit block named a deleted file, so `git add` added nothing. Never name deleted files in `git add`.
- An approved result (Golf It "6 dots tall, 2 apart") was generalised away by an automatic method. An approved result is a regression test.
- Pablo showed an example to follow ("crisp: 4 amber tones") and it was rewritten from scratch, which broke it. Reproduce an example exactly first, then change one thing at a time beside it.
- Experiments were started inside the app code. Experiments go in the lab (a copy); only the chosen result is ported.
- Pablo was told to run a build from the repo folder; he launches Quarter by typing `quarter` (`go install ./cmd/quarter`).
- Agents' self-reports claimed wins their sheets didn't show. Judge every sheet by eye.
