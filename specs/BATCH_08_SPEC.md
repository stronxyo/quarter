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

## Phase A2: Screen layout, anchored left and right (done 2026-09-28; Pablo: "now it looks perfect")

Pablo, 2026-09-28: at a normal zoom the screen looks right (quarter_ box and list on the left, logo and info on the right); zoomed out, everything sits in the left half and the right half is empty. He wants the quarter_ box and the names to stay on the left, and the logo and info always on the right, like cliamp (left block pinned left, right block pinned right, lines stretching between).

### What happens today (read from the code)

- The list is capped at 80 columns (`maxTableWidth`), so the numbers stay near the names. The info pane grows from 40 to 68 columns (`paneMinWidth`, `paneMaxWidth`). The right column starts where the list ends, so the whole layout stops at about 150 columns. On a 240-column window (zoomed out), 90 columns on the right are empty.
- The logo band starts after the quarter_ box (column 82), but the info pane starts after the list. When the list shrinks below 80 columns (windows 98 to 121 wide), the logo and the pane don't line up.
- Zoom is the terminal's font size, set by each user; no app can change it. A zoom arrives as a new window size in columns and rows.
- **Found while planning:** Quarter asks the terminal for its cell size in pixels once, at start. After a zoom, pictures keep the old pixel size, too big or too small for the band.

### The layout

W is the window's width in columns.

- **Left block, pinned to the left edge:** the quarter_ box (as now, 2 columns in) and under it the list. The list keeps its readable width: at most 80 columns, and 56 at the least before the pane becomes a pop-up (as now).
- **Right block, pinned to the right edge:** the logo band and under it the info pane, one column, the same width R, left edges lined up. R = W - list - 2, kept between 40 and 80 columns (was 68; see "Decided from common practice").
- **The gap between them** is whatever is left, at least 2 columns (as now). On big windows it grows; nothing is centred, so the left block never moves.
- **The section line** (`▸─ [All]  Installed  Recent  Favourites ── 300 games ──`) runs from the left edge to 2 columns before the right block, bridging the gap as cliamp's lines do. The column headers and the rows stay within the list's 80 columns.
- **Logo band:** shown when the quarter_ box and the right block both fit: W at least 82 + 40 = 122, and the box's own height rules as now. Between 106 and 121 today a narrow band (24 to 39 columns) shows beside the box, not lined up with the pane; it goes, and the pane starts right under the box.
- **Heights:** unchanged. The list fills down to the key help; the pane is as tall as the list.

### At three sizes

```
~100 columns (normal zoom, narrow): no logo band; list 58, pane 40 pinned right
 ┌ quarter_ ────────────────────┐
 └──────────────────────────────┘
 ▸─ [All] Installed ── 300 games ──  ╭ ROCKET LEAGUE ────╮
   Name            Played  Last      │ Psyonix · 2015     │
 ✦ Rocket League   2712h   today     │ ...                │

~160 columns: list 80, gap 2, right block 78, like Pablo's reference screenshot
 ┌ quarter_ ───────────────────────────────────┐  ROCKET LEAGUE (picture)
 └─────────────────────────────────────────────┘
 ▸─ [All] Installed Recent Favourites ── 300 ──  ╭ ROCKET LEAGUE ───────────╮
   Name                     Played  Last played  │ ...                       │

~240 columns (zoomed out): list 80, right block 80 at the right edge, gap 80
 ┌ quarter_ ─────────────────────┐                                  ROCKET LEAGUE (picture)
 └───────────────────────────────┘
 ▸─ [All] ... ── 300 games ──────────────────────────────────  ╭ ROCKET LEAGUE ─────╮
   Name                Played  Last played                      │ ...                 │
```

### Revision after Pablo saw it (2026-09-28): no empty centre

Pablo: "looks really good, but I don't like so much how that empty center is seen". Changed, option A of two:
- **The pane fills the right side:** it starts 2 columns after the list and runs to the right edge at every width; no gap, no 80-column cap. The logo band is the pane's width, so the logo (picture or text) is centred over the pane.
- **Its contents are a centred column at most 76 characters wide** (title, info rows, description): readable lines (see above), on the same centred axis as the logo. Up to a pane 80 wide, exactly as before.
- Option B, text across the whole pane, was not chosen: very long lines on big windows, and text off to the left of a centred logo.
- Then, Pablo after seeing it: the column goes to the pane's left edge, wrapped the same (76 at most). The logo stays centred over the pane.

### Unchanged

- Narrow windows: under 98 columns the pane is a pop-up (`d`), as now; under 80 no quarter_ box, the compact badge instead.
- `d` hides the pane on wide windows (the list takes the left as now; no right block, no logo band). `o` hides logos (the right block keeps the pane).
- The key help and the footer stay on the left.

### Pictures and zoom (fixes)

1. **Ask the cell size again on every window size change,** and draw pictures at the size it reports. Until the answer comes (or if none comes), the old size is used, capped so a picture never spills out of the band.
2. **Resizing by dragging:** each new band width is a new picture. Pictures are drawn only once the size has settled (the existing debounce), and pictures for other sizes are dropped from the cache, so dragging doesn't build up dozens of pictures in memory.
3. The picture is redrawn when the right block moves (it already follows the band's position and size).

### Tests (before Pablo sees it)

- A table of window sizes, 60 to 260 columns wide and 20 to 70 rows tall, checking on the drawn screen: the quarter_ box starts at column 2; the pane's right border is the window's last column; the logo band and the pane start at the same column; the list is at most 80 wide; the gap is at least 2; no line is wider than the window; the section line ends 2 columns before the right block.
- The same with `d` (pane hidden), `o` (logos hidden), a search in progress, the full key help (`?`), and fewer games than rows.
- Pictures: after a size change the cell size is asked again, and the picture's pixel size follows the reply; pictures of old sizes are dropped.
- Then by eye in Windows Terminal at three zooms, screenshots to Pablo before the commit.

### Decided from common practice (Pablo left both to research, 2026-09-28)

- **The right block's widest: 80 columns, not 84.** Text inside the pane is then 76 characters a line. Readable line length is 45 to 75 characters, 66 the most cited ideal (Bringhurst, via [Baymard](https://baymard.com/blog/line-length-readability) and [Wikipedia](https://en.wikipedia.org/wiki/Line_length)); WCAG 1.4.8 sets 80 as the upper limit ([W3C](https://www.w3.org/WAI/WCAG22/Understanding/visual-presentation.html)). 80 also mirrors the list's 80, so the two blocks are the same width at their widest. The logo picture loses nothing: at 13 rows tall a typical logo is limited by its height well before 80 columns.
- **The list stays at 80 on wide windows; the gap takes the rest.** In a table, text is left-aligned and numbers right-aligned so the eye can compare them, and spreading columns apart makes rows hard to follow ([A List Apart](https://alistapart.com/article/web-typography-tables/), [Pencil & Paper](https://www.pencilandpaper.io/articles/ux-pattern-analysis-enterprise-data-tables)). Material Design's list-detail layout keeps the list a fixed width and lets the detail grow ([Android Developers](https://developer.android.com/develop/adaptive-apps/guides/canonical-layouts)); here the detail is capped too, by line length, so what's left is a gap. lazygit instead splits by fractions (`sidePanelWidth: 0.3333`, [docs](https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md)), which suits its wide diffs; Quarter's rows are a name and two short numbers, so wider rows only add empty space inside them.

## Phase A3: UX (paging and INSERT QUARTER built 2026-09-28; GAME OVER open)

PLAN.md's Phase A, with Pablo's decisions of 2026-09-28.

### Decided

- **`?` help:** stays as it is (the key help at the bottom expands to every key). No pop-up overlay. Done.
- **Turning logos off:** `o` already does it. No separate setting for pictures versus text blocks; added later only if someone asks.
- **Paging wraps:** `→` on the last page goes to the first, `←` on the first page to the last.
- **Moving runs on across pages** (Pablo changed it, 2026-09-28: better UX): `↓` on a page's last game goes to the next page's first, `↑` on its first game to the previous page's last; past the very ends it wraps round, like `←` `→`. In `internal/ui/navigate.go`.
- **The look for the start and the end:** "a PC machine mixed with arcade", not full arcade.

### Start: INSERT QUARTER (first run only)

- A short boot screen in the spirit of a 90s PC checking its hardware, with arcade words, for example `QUARTER BIOS v0.1 · scanning libraries... 300 games OK`, then the big quarter_ logo with `INSERT QUARTER` blinking under it. Any key starts.
- Only on the very first run; afterwards Quarter opens straight to the list. Built: the setting `splash_seen` in `config.json` is saved when a key leaves the splash (so anyone upgrading sees it once too); `main.go` passes `Splash: !splash_seen`. `q` and `ctrl+c` still quit from it; no game logo is drawn over it. Code in `internal/ui/splash.go`.
- **Built 2026-09-28 (Pablo: "agree in everything, so go"):** `internal/intro` holds the start screen and all 32 intros (ported from the lab unchanged but for one dot character not every font has). On every start a random intro, never last start's (`last_intro` in config.json), leads into the start screen; any key skips it, the next key starts (q counts as any key there, Pablo: it mustn't quit by habit; only ctrl+c quits from the start screen); `"intro": "off"` or an intro's key (`"crt-power-on"`) in config.json. Beside the double-size logo (windows about 144 x 38 and up), "insert quarter" in the logo's own lowercase letters at half size; on smaller windows the normal logo and a line of text; below the box's size, a compact screen with the badge. Tests: every intro at three sizes stays in the window, uses only common characters and ends with at least 80% of the logo in place; picking never repeats; skip, off, named, resize, quit.
- **Renamed and redesigned 2026-09-28** (no trademarks or characters from known games; their gameplay ideas are free to use): Rally (paddles rally a coin), Coin roll (a spinning quarter, no mouth or ghost), Block drop (plain column bands, no tetromino shapes), March (the logo marches in step, no ship or shots).
- **Intros (lab, 2026-09-28):** Pablo loved the intro lab ("love them all"): 32 intro animations in `Documents\Quarter\Intro lab\` (run `quarter-intro`), researched and sourced in its `research\RESEARCH.md`. Planned for the app after his review: a random intro on every start, any key skips it; the logo at double size on big windows (built from the art, shadow lines kept thin); INSERT QUARTER as one smaller spaced line ("the title needs to be the principal"). Not in the app yet.
- **It shows on every start** (Pablo, 2026-09-29, final: "lets leave it always, i like it"). `splash_seen` is still saved, unused for now.
- The boot lines show the real scan as it happens, so the splash never adds waiting of its own.

### End: the goodbye (built 2026-09-28)

Pablo's pick after a brainstorm (he liked none of "game over", "terminal over", "game finished"): 11 lines, one at random each quit, never last quit's (`last_goodbye` in config.json): keep the change, quarter returned, out of quarters already?, thanks for the quarter, shutting down..., session saved, it's now safe to close your terminal, thanks for playing, high score saved, powering down..., see you next round. No fixed pairs.

- Printed into the shell once Quarter has closed, instantly: the line in the logo's lowercase letters at the size of "insert quarter" (16 new letters and ? . ' drawn the logo's way), one column in from the left edge (Pablo, after seeing it flush left); plain text when too wide.
- 2026-09-29: a centred goodbye inside Quarter before it closes was discussed and dropped; Pablo liked the line staying in the shell. Moved from centred to the left (his pick A); a smaller 3-row cut of the letters (B) was tried in the lab and not taken.
- Under it, only when a game ran while Quarter was open: `played this session: 2h 10m` (Steam's running game, polled every 2 s; a game still running at quit counts).
- `"goodbye": "off"` or a goodbye's key (`"keep-the-change"`) in config.json.
- Code: `internal/intro/goodbye.go`, `internal/ui/goodbye.go`, `sayGoodbye` in `cmd/quarter/main.go`. Checked on the real program loop in a lab copy: quitting from the list returns the final state, the goodbye prints and is saved.

### Playtime after a game stops (built 2026-09-29)

Pablo: when a running game stops, its played time on Steam changes, so Quarter must check it every time.

- Checked on his machine first: Rocket League closed at 02:48:29 and Steam wrote the new `Playtime` and `LastPlayed` to `localconfig.vdf` at 02:48:43, with Steam still running. So the numbers land about 15 s after the game stops, not only when Steam closes.
- When the running game (polled every 2 s) stops or changes, Quarter re-reads the library 10, 20, 40, 70 and 130 s later, stopping as soon as that game's numbers move. After the last check it gives up; r still rescans.
- A rescan already running (r) is waited for, not doubled, and doesn't count as a check. A second game stopping takes over the wait: any re-read picks up both.
- Code: `internal/ui/afterplay.go`. Checked on the real program loop in a lab copy with the real Steam folder: the stop is seen and every check runs and ends.

### End: GAME OVER (superseded by the goodbye above)

- Quitting stays instant: no delay or animation before the terminal is back. The goodbye is printed into the shell after Quarter closes.
- Options to mock up for Pablo when he's ready, all in the PC-plus-arcade mix:
  1. `GAME OVER · CREDITS 0` and then the prompt, like a DOS machine after the game ends.
  2. `Saving high scores... done. It's now safe to close your terminal.` (a nod to the Windows 95 shutdown screen).
  3. A one-line session log: `PLAYER 1 · 2712.3h on record · last: ROCKET LEAGUE · see you soon.`
- Possibly a setting for no goodbye at all, for people who want a clean shell.

### Tests

- Paging: wraps both ways; a list of one page; after a search; `g`/`G` unchanged.
- Moving: wraps within the page at both ends; on the last, shorter page; with a single game.
- Splash: shows on first run only, any key leaves it, never delays the list when the scan is quick.
- Goodbye: printed after the screen is restored, one short block, nothing when turned off.

### Phase A mistakes, so they aren't repeated

- Judging by brightness alone erased dark logos (Cyberpunk). Judge by transparency; only near black is a gap.
- Tuning on one or two logos broke others. Always check the whole test set.
- A commit block named a deleted file, so `git add` added nothing. Never name deleted files in `git add`.
- An approved result (Golf It "6 dots tall, 2 apart") was generalised away by an automatic method. An approved result is a regression test.
- Pablo showed an example to follow ("crisp: 4 amber tones") and it was rewritten from scratch, which broke it. Reproduce an example exactly first, then change one thing at a time beside it.
- Experiments were started inside the app code. Experiments go in the lab (a copy); only the chosen result is ported.
- Pablo was told to run a build from the repo folder; he launches Quarter by typing `quarter` (`go install ./cmd/quarter`).
- Agents' self-reports claimed wins their sheets didn't show. Judge every sheet by eye.

## Play sessions, recorded (Pablo, 2026-09-29: "lets start saving and recording the play sessions")

The session history screen comes after v0.1 (`PLAN.md`); recording starts now so there is data when it comes. No screen yet.

### The source: Steam's own log

Steam writes every game start and stop to `<Steam>\logs\gameprocess_log.txt`, whether or not Quarter is open. Read on Pablo's PC (read-only, 2026-09-29):

- It goes back to 2026-03-21 and survived 21 Steam client updates: 294 sessions across 11 games, about 540 KB, CRLF lines like `[2026-03-21 23:02:25] AppID 252950 adding PID 23208 as a tracked process "..."` and `[2026-03-21 23:15:32] Remove 252950 from running list`.
- Times are local, with no offset. Each game's last `Remove` lands 0 to 2 s before `LastPlayed` in `localconfig.vdf` (all 11 games checked).
- For games whose whole history is inside the log, its minutes match Steam's totals (16/16, 8/8, 6/5; one 6-second session shows 6 minutes on Steam).
- Other projects read the same file for sessions ([SteamPlaytimeTracker](https://github.com/12Acorns/SteamPlaytimeTracker)).
- Unverified: when Steam rotates it (other logs there roll to `.previous.txt` at 512 KB to 8 MB; this one hasn't at 540 KB), and whether Steam writes it on Linux and the Steam Deck.

### How a session is read

- A session opens at a game's first `adding PID` line and closes at its `Remove <appid> from running list` line. Sessions are per game, not per process: a game can run up to 7 processes (launchers, overlays), and two games can overlap.
- A `Client version` line is Steam starting: any session still open there had no clean stop (Steam crashed or the PC shut down; 3 on Pablo's PC). It is kept with its end unknown and counts no time, never the hours until Steam started again.
- A `Remove` with no start is ignored (2 on Pablo's PC). A session still open at the end of the log is a game running now: it is recorded on a later read, once it has stopped.
- `gameprocess_log.previous.txt` is read first when it exists, in case the log has rotated.

### How it is kept

- `sessions.json` in Quarter's data folder (with `library.json`), so history survives if Steam clears or rotates its log. Written like the other files: a temporary file, then a rename.
- Each session: the game, its start, its end (UTC), and whether the end is known. A session is known by game and start, so reading the log again adds nothing twice.
- Read after every library scan (at start, `r`, and the checks after a game stops), off the UI thread. Failing to read the log or save the file is logged, never shown.
- Sessions under 60 s are kept in the file; stats can leave them out later.

### Tests

- The reader on made-up logs: clean sessions, several processes, overlapping games, a crash before a restart, a stray `Remove`, a game still running, CRLF, a rotated `.previous` file.
- Merging: nothing added twice, new sessions added, order by start.
- By hand, read-only: Pablo's real log gives 294 sessions.

## Session history (Pablo, 2026-09-29: moved before Phase C; "if all is ready to implement go")

`PLAN.md` had the history screen after v0.1, to wait for weeks of real data. Reading Steam's own log gave six months at once (298 sessions on Pablo's PC), so it moved into Batch 8. Chosen in the history lab (`Documents\Quarter\History lab`, run `quarter-history`) out of five layouts: his mix of the ranking (C) and the board (D), both in the main screen.

### How it works

- **`h` cycles three views:** the games (as now), then the ranking, then the board, then the games again. `esc` goes straight back to the games.
- **Ranking (first `h`):** the game list becomes MOST PLAYED: games ranked by time played in the period, with a bar each. The section line holds the periods: 7 days, 1 month, 3 months, 6 months, 1 year (no "since", Pablo). The keys work as in the games (Pablo, 2026-09-29: "in games we have another concept"): `tab` moves on to the next period, wrapping round, as it moves through the filters; `←` `→` turn pages, round past either end, keeping the row where the page has one; `↑` `↓` run on across pages and round past either end; `g` `G` jump to the first and last. Pages are the game list's length, with the same `2/3` counter.
- **The pane beside the ranking:** the ranked game's history: its time in each of the five periods, `Total` (the game's lifetime playtime from its launcher, Steam today; launcher-neutral words, Pablo: "we will have other platforms"), longest session, streak (days in a row, and best), the last 16 weeks as bars, the weekdays as bars, the 6 latest sessions.
- **Board (second `h`):** the ranking stays; the pane becomes HIGH SCORES: the period's top 8 like an arcade board (`1ST ROCKET LEAGUE....... 83.0H`), a year of played days as a calendar (a column per week, Monday on top, amber shades for how long), and records: total for the period, longest session, streak, the weekday played most. The ranking and the board's top 8 both stay (Pablo: "leave them"). When the pane is short (the logo band takes its rows on smaller windows), the calendar and the records always show and the places give way, down to a podium of 3.
- **Empty days and weeks** are the achievement bar's thin dim line (`─`), not a new grey (Pablo picked it over plain black in the lab).
- **The logo band and the pane follow the ranked game**, not the list's selection. While history is open, only its keys, `r`, `?` and quitting work; play, install, favourite, search and the others wait for the games view, so a key never acts on a game that isn't the one on show.
- **Narrow windows** (no pane beside the list): the ranking takes the list's place, and the board takes it on the second `h`.
- **Live:** Quarter loads `sessions.json` at start, and every time new sessions are saved (after each library scan: at start, `r`, and the checks after a game stops) the ranking and the board update.

### Counting

- Sessions under a minute, and sessions with no known end (Steam stopped first), count no time. Steam tools (SteamVR, Proton) are left out of the ranking.
- Periods end today and count whole days back (7 days is today and the 6 before). Weeks run Monday to Sunday. A session counts on the day it starts, in the PC's time zone.
- History starts where Steam's log starts (2026-03-21 on Pablo's PC); `Total` is the launcher's lifetime figure.

### Code

- `internal/history`: the counting, from sessions to periods, weeks, days, weekdays, streaks and records. Knows nothing about the screen.
- `internal/ui/history.go`: the three views and their keys. Colours in `styles.go`, the key in `keys.go`.
- `cmd/quarter/main.go`: loads `sessions.json` for the UI; the recorder hands the full list back after each save.

### Tests

- History: every figure on made-up sessions (periods, top, weeks, days, weekdays, last, longest, streaks), short and unknown sessions not counted, days in the local time zone.
- UI: the `h` cycle and `esc`; the periods wrap round and leave out "since"; moving like the game list (wrap, pages, `g` `G`) on 40 games; the pane and the logo follow the ranked game; other keys wait; narrow windows; no sessions at all; new sessions arriving; every view fits the window.
- The real program loop in a lab copy, with Pablo's real data read-only, before installing.

## Keys that follow the game (Pablo, 2026-09-29: "if a game is installed, the i must disappear and do nothing"; "go with a")

Until now the key line was the same for every game: `enter play` even on a game that wasn't installed (where Enter installed it), and `i install` even on an installed one (where it sent Steam's install link anyway).

### What each key does, by the selected game

| Game | Key line | Keys |
|---|---|---|
| Installed | `enter play  u uninstall` | `i` does nothing |
| Not installed | `enter/i install` | both install; `u` does nothing |
| Running | `▶ running` | no game key works on it (play, install, uninstall, store page): Pablo, "an exit one will stop the process". Enter says it's running. Favourite still works (it changes nothing in Steam). |
| Updating | `↓ updating` | Enter says it's updating; `i` and `u` do nothing; the store page works |
| None (empty list) | no enter, i, u, s, f | nothing to act on |

- **Uninstall** (`u`) sends Steam's uninstall link, `steam://uninstall/<appid>` (community documented: [Steam forum](https://steamcommunity.com/discussions/forum/1/540744935066349428/), [ReVuln](https://revuln.com/files/ReVuln_Steam_Browser_Protocol_Insecurity.pdf)). Whether Steam asks before uninstalling is unverified, so Quarter asks first: the first `u` shows `Press u again to uninstall <game>` in light red in the message line (under the details pane, or at the bottom on narrow windows); a second `u` within 3 seconds sends it. The message going (3 seconds) or moving to another game cancels it. A separate key, not `i`, so a habit press of `i` can never uninstall.
- **Search (option A):** while typing a search, Enter plays or installs the game under the cursor at once (before, it only confirmed the search). The key line shows the game's Enter and `esc clear`. `i` can't install while typing: the letter goes into the search.
- **Store page** (`s`) on a game the store has no page for says so instead of opening a broken page.
- **Logos** (`o`) on a window too narrow for the logo band says "Logos show on wider windows" instead of changing a setting nothing shows.
- **The pop-up pane** on narrow windows shows `d/esc close` in the key line.
- **`?`** lists every key (`enter play/install`, `i install`, `u uninstall x2`, the rest), and adds the page and end keys the list already had (PgUp/PgDn, Home/End), short enough to fit 80 columns. The key line shows only what the selected game allows.
- **The key line always fits:** as the window narrows, the arrows give way first, then `h history`, then the sort keys, then keys from the middle; the game's keys and `q quit` stay. (bubbles v2.2.1's help doesn't cut reliably: an item that overshoots with no room for "…" is added anyway, so Quarter cuts the line itself.)

### Tests

- Each game state: the key line, and every game key doing only what the table says.
- Uninstall: one `u` only warns (red, nothing sent); two within the window send the link; the timer or moving away cancels; nothing on a running, updating or not installed game.
- Search: Enter while typing plays an installed game and installs one that isn't; `esc` clears.
- The store page on a game not on the store, `o` on a narrow window, the pop-up's key line, the `?` list.
- A case in `quarter-errors` to try each state by hand.

## Phase B: Error states (approved 2026-09-29, "go")

Every case below was researched (sources at the end) and is tested on fake Steam folders in `testdata/`, never Pablo's real Steam.

### B1. Steam not found

- **More places:** Windows adds `HKLM\SOFTWARE\WOW6432Node\Valve\Steam` `InstallPath` after `HKCU` `SteamPath` (both exist on Pablo's PC). Linux adds Flatpak (`~/.var/app/com.valvesoftware.Steam/.local/share/Steam`), Snap (`~/snap/steam/common/.local/share/Steam`, `~/.snap/data/steam/common/.local/share/Steam`) and `~/.steam/root`, as protontricks does.
- **The screen:** "Steam not found", every place Quarter looked, and what to do: install Steam, or start with `--steam-path` or `QUARTER_STEAM_PATH`. `r` looks again (Steam installed while Quarter is open).
- A `--steam-path` (or `QUARTER_STEAM_PATH`) that isn't Steam says so: that folder has no `steamapps` folder.

### B2. No games, no libraries

- **Empty lists** say why, instead of bubbles' `No items.`: nothing installed ("No games installed yet. Install one in Steam, then press r."), Favourites ("No favourites yet. Press f on a game to star it."), Recent ("Nothing played in the last 14 days."), a search ("No games match.").
- **A library on a drive that isn't connected:** `libraryfolders.vdf` keeps listing it with its apps, so the footer says "Library D:\SteamLibrary not connected, 5 games hidden". They come back with `r` once the drive is back.
- **Steam never signed in** (no `loginusers.vdf`, or no account in it): not a damaged file; the footer says "Sign in to Steam once to see playtime."
- **Games Steam is updating** (Pablo, 2026-09-29: "a blue text saying updating similar to the green running"): until now a game mid-update was dropped from the list, since only fully installed games were read. Now a game with any of Steam's update flags and not the fully installed one (update required 2, running 256, paused 512, started 1024, validating, adding files, preallocating, downloading, staging, committing; [Lutris: Steam StateFlags](https://github.com/lutris/lutris/blob/master/docs/steam.rst)) is listed, its row in blue (`#89B4FA`, the running green's neighbour in the same palette) with `↓ updating` where the last played date goes, and `↓ UPDATING` in the pane. A playable game with an update waiting (6) stays a normal row. A first download uses the same flags (unverified), so it shows as updating too.
  - Enter on it says "<game> is updating in Steam" and starts nothing (what Steam does when asked to play a game mid-update is unverified).
  - While any game is updating, the library is read again every 30 s, so the blue clears by itself when Steam finishes.
  - `↓` is U+2193, in Cascadia Mono and JetBrains Mono (checked in their character maps). The install message's `⬇` (U+2B07) is in neither font, so it became `↓` too.

### B3. Offline

Network failures fall into four kinds; only the last is remembered:

| Kind | Seen as | What Quarter does |
|---|---|---|
| No network | name lookup, connection or timeout failure | Store and logo downloads pause; "offline" shows dimly beside the game count. One request is tried after about 15 s, then 30 s, 60 s and so on, at most 5 min apart, with random spread ([AWS](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)). Any answer from the server means the network is back: the marker goes and downloads resume, the selected game first. |
| Rate limited | HTTP 429 or 403 from the store | Store requests pause (logos come from another server and go on), for Retry-After if the store sends it, otherwise from 60 s, doubling, at most 10 min. The pane says "store busy, retrying". |
| Server error | other HTTP failures (5xx) | Paused and tried again like no network, without the offline marker. Never remembered as a failure. |
| Not there | store `success:false`, CDN 404 | Remembered, as today. |

- No failure blocks a game for the whole run any more (today one failed request did, for its store info and its logo).
- Calm wording, dim, no red, no pop-up, no raw error on screen; raw errors go to `quarter.log`.

### B4. Damaged files

- **`libraryfolders.vdf` unreadable:** Steam keeps a second copy in `config\`; it is tried before falling back to the install folder only.
- **An empty or unreadable Steam file:** read once more after 100 ms, since Steam may be writing it. An empty file counts as unreadable. If it still fails, only that file is skipped.
- **The footer names it** when one file failed ("Steam's file for app 620 (appmanifest_620.acf) can't be read, so that game is hidden"; the playtime and library files the same way), and counts them when several did. Details in `quarter.log`.
- **Quarter's own files on Windows:** saving retries the rename for about a second when antivirus or the file indexer holds the file ([golang/go#29106](https://github.com/golang/go/issues/29106)). Already handled: damaged caches are ignored, a damaged `config.json` is moved aside.

### How messages look (Pablo, 2026-09-29, after seeing them in the error lab)

- **Light red (`#FF7B72`) for what is wrong**, so it stands out from the amber: the "Steam not found" screen's title and the reason, `offline` beside the game count, "store info unavailable, retrying" and "store busy, retrying" in the pane, a library not connected, damaged Steam files, a failed rescan, "Could not open Steam". Guidance stays dim (empty lists, "Sign in to Steam once to see playtime"); action messages stay amber.
- **Under the details pane**, not under the key help: when the pane is beside the list, it gives up as many rows as the message needs (wrapped to the pane's width, at most 3 lines) and the message sits under its border. Without the pane beside the list (narrow windows), it stays the line under the key help.
- Seen first in the error lab (`Documents\Quarter\Error lab`, run `quarter-errors`): ten error screens on fake Steam folders.

### Tests

- Steam not found: every candidate missing, a bad override, `r` finding Steam afterwards.
- Empty lists: each message, in each filter and a search.
- Missing drive: a fixture library entry pointing at a folder that doesn't exist, with its app count.
- Never signed in: no `loginusers.vdf`.
- Offline: each kind from a local test server (`httptest`) and an address that can't be reached; pause, marker, probe and recovery through the model's `Update`.
- Damaged: an empty manifest, a broken `libraryfolders.vdf` with a good `config\` copy, a file fixed between the two reads.
- The screens once more through the real program loop in a lab copy before installing.

### Sources

- [protontricks steam.py](https://raw.githubusercontent.com/Matoking/protontricks/master/src/protontricks/steam.py) (Linux install places)
- [AWS: Exponential Backoff And Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Go net package](https://pkg.go.dev/net) (DNS and connection errors, timeouts)
- [Steam forum: 429 Too Many Requests](https://steamcommunity.com/discussions/forum/1/3492005739881311668/), [apis.io: Steam rate limits](https://apis.io/rate-limits/steam/steam-rate-limits/) (community sources: about 200 requests per 5 min, 429 when over, Retry-After not always sent)
- [clig.dev](https://clig.dev/), [NN/g: error message guidelines](https://www.nngroup.com/articles/error-message-guidelines/) (say what happened and what to do, no blame, keep the user's data on screen)
- [golang/go#29106](https://github.com/golang/go/issues/29106) (rename failures on Windows)
- [steam-for-linux#9672](https://github.com/ValveSoftware/steam-for-linux/issues/9672) (no running game on Linux)
