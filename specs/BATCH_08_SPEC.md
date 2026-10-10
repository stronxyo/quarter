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

## Phase A3: UX (built 2026-09-28: paging, the start screen, the goodbye)

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

### Tests

- Paging: wraps both ways; a list of one page; after a search; `g`/`G` unchanged.
- Moving: wraps within the page at both ends; on the last, shorter page; with a single game.
- Start screen: shows on every start, any key leaves it, never delays the list when the scan is quick.
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
- **Ranking (first `h`):** the game list becomes MOST PLAYED: games ranked by time played in the period, with a bar each. The section line holds the periods, where the games have their filters: 7 days, 1 month, 3 months, 6 months, 1 year (no "since", Pablo), and **All time** (Pablo, 2026-09-29: he has played since 2010, Steam's log on this PC only reaches back to March 2026): All time ranks every game by its lifetime playtime from its launcher, exact back to the account's first day, games without a recorded session included; the board's TOTAL is then the lifetime sum. MOST PLAYED heads the ranking's name column (the six periods fill the section line). The keys work as in the games (Pablo, 2026-09-29: "in games we have another concept"): `tab` moves on to the next period, wrapping round, as it moves through the filters; `←` `→` turn pages, round past either end, keeping the row where the page has one; `↑` `↓` run on across pages and round past either end; `g` `G` jump to the first and last. Pages are the game list's length, with the same `2/3` counter.
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

## Past years from Steam (Pablo, 2026-09-29: "past years it is go")

Pablo has played since 2010, but Steam's log on this PC starts on 2026-03-21 (Windows reinstalled 2026-03-10). Steam Replay, Valve's yearly recap, holds each year's playtime per month since 2022. Researched and tested (read only) on his account, then tried in the Replay lab (`Documents\Quarter\Replay lab`, `quarter-replay`). Named "past years" in Quarter (Pablo: "replay feels like re watching something"); the question names Steam Replay once, since that's where each year is set to Public.

### The source

- Steam's Web API method `ISaleFeatureService/GetUserYearInReview/v1/?steamid=<id>&year=<year>`, the one Steam Replay's page reads. Defined in Steam's protobufs ([SteamDatabase: service_salefeature.proto](https://raw.githubusercontent.com/SteamDatabase/Protobufs/master/webui/service_salefeature.proto)), not in Valve's documentation: Quarter treats it as optional and fails quietly.
- Only the user's public SteamID (read from `loginusers.vdf`, as for playtime); no login, no key, no token. Answers for a year set to Public; a private year comes back empty (`{"response":{}}`).
- Per year: `playtime_stats.months[]`, each with the month's start (`rtime_month`, UTC midnight on the 1st), total seconds and sessions, and per game (`appid[]`). Tested on Pablo's account: 2022 240.5 h, 2023 211.2 h, 2024 163.2 h, 2025 134.0 h. 2023 has monthly totals but no per-game months (per game only for the year). The current year isn't there until Valve publishes it in December; a year ends around December 14 (published reports).

### Opt-in (Steam Web API Terms: data only "as requested by the end user", users told what's stored)

- Asked for only from the history views, never at start (Pablo: "i dont want to be intrusive"). The game's history pane ends with `Older years? a add your past years from Steam` until they're in. The history key line shows `a past years`; `?` lists `a past years (in history)` (`a years` on windows under 100 columns, where the `?` view is full).
- `a` shows the question in the pane: what is fetched (the yearly recap, 2022 to last year, per game per month), nothing else and no login, saved only on this PC (`replay.json`), each year must be Public in Steam Replay, `a` later removes them. `y` brings them in, `esc` declines.
- The answer in the message line: `Past years added: 2022, 2023, 2024, 2025`, naming years Steam didn't give (not public). None at all says how to make them public. `a` again asks to remove them; `y` deletes `replay.json`.
- The privacy note for release (Phase D) says what's fetched and stored.

### In history

- A Replay month counts only if it began before the month of the first recorded session, so no time counts twice.
- The calendar shades those months: the month's time spread over its days, as a smaller square (`▪`), with `▪ month average, past years` in the legend.
- Periods include Replay months that begin inside them (1 year now reaches back into last year's months).
- The board gets a YEARS line: each year's time, Replay and recorded together (`2022 241h · 2023 211h · ...`).
- 2023's months have no per-game split, so they count in totals, the calendar and YEARS, not in the ranking.

### Code

- `internal/replay`: the client (`Fetch`) and the answer's parsing; `ErrNotPublic` for an empty year.
- `internal/history`: `WithReplay`, `Months`, `Years`; `Played` and `Top` include Replay months.
- `internal/ui/replay.go`: the opt-in, the question, the calendar's month shading, the YEARS line.
- `cmd/quarter/main.go`: loads `replay.json`, saves it after the fetch, deletes it on removal; the fetch asks each year from 2022 to last year for the account signed in on this PC.

### Tests

- The parsing on recorded answers (a full year, 2023's shape, a private year): no live calls in tests.
- History: Replay months before the first recorded month only, periods, top, months, years.
- UI: the hint only in history and only until added; `a` in the games does nothing; the question, `esc`, `y` with the fetch's result saved; years not public named; removal deletes; the calendar's `▪`; 1 year including Replay months; the YEARS line; the key line and `?` at 80 and 100 columns.
- The real program loop in a lab copy with Pablo's real account, the file saved to a temporary folder.

## Download progress (Pablo, 2026-09-30: "the blinking arrow is nice, and go implement")

Games Steam is installing or updating were all `↓ updating`, with no progress. Now installing and updating are told apart, and both show their progress, in the same blue. Tried in the download lab (`Documents\Quarter\Download lab`, `quarter-downloads`) on a simulation and checked against a real install.

### Where progress comes from (measured 2026-09-29 on DiRT Rally 2.0, 115.2 GB)

- The game's install file (`appmanifest`) has the download's bytes, but Steam writes it only when a download starts, pauses or ends: it said 9% for the whole half hour. Its `buildid` is 0 until a first install finishes, which tells installing from updating; StateFlags 512 marks a pause.
- The download folder is preallocated at full size, so its size never moves.
- Steam's download log, `logs/content_log.txt`, writes `update started : download X/Y` at the start, the phase as it changes (`App update changed : Running Update,Downloading,Staging,`, then Staging and Committing), `Current download rate: N Mbps` about once a minute (the client's total), and `finished update` / `scheduler finished` at the end.
- Adding up the minute rates from the starting bytes gave 104.1 GB against the real 104.7 GB (99%). Replayed through the reader up to mid-download: 90%, 103.4 of 115.2 GB, 65 MB/s, the same as by hand.
- Researched: the exact live figures exist only inside the Steam client (its `SteamClient.Downloads` interface), reachable from outside only through its unauthenticated debugging port or undocumented native interfaces. Not used: the port lets anything on the PC run code in Steam's window while open, and both break with Steam updates ([decky-frontend-lib](https://github.com/SteamDeckHomebrew/decky-frontend-lib/blob/main/src/globals/steam-client/Downloads.ts), [steam-debug](https://github.com/kmturley/steam-debug), [open-steamworks](https://github.com/SteamRE/open-steamworks)). No launcher found shows live Steam progress.

### What shows

- **The row's last column** (13 wide): `↓ ~42%` while downloading, `↓ applying`, `↓ verifying`, `↓ paused`, `↓ queued` (another game is downloading), or `↓ installing` / `↓ updating` before the log says more.
- **The pane:** `↓ INSTALLING ~42%` (or UPDATING), a blue bar (the rest the thin dim line), and `48.3 of 115.2 GB · 65 MB/s · about 17 min left`; for the other phases a line saying what Steam is doing (`Downloaded. Steam is putting the files in place.`, `Paused in Steam.`, `Waiting in Steam's download queue.`). A paused download keeps its percentage: `↓ UPDATING ~50% · PAUSED`.
- **Motion:** the arrow blinks (once a second; twice until C3, see there) while Steam is working on it: downloading, applying or verifying (Pablo's pick of four in the lab: dots, spinner, blinking arrow, moving bar). Paused and queued stay still.
- `~` marks the estimate. Between log lines the percentage carries on at the last speed, every half second, for at most 90 s past the last line (Steam logs about once a minute), so a stalled download never counts itself up.
- While two games are marked downloading, the rate can't be split and neither advances (Steam downloads one at a time).

### Watching

- Every 2 s Quarter checks whether the download log changed (its size), and reads it only then; while a game downloads it also reads those games' install files (a pause shows only there).
- A download the log shows for a game the library doesn't know as downloading (started while Quarter is open, or a new install) reads the library at once. A download that ended does too, so the game turns playable within seconds.
- Unverified: that Steam marks a pause in the install file the moment it's paused (DiRT's install never paused).

### Code

- `internal/steam/downloads.go`: `ReadDownloads` (the log's last 4 MB) and `Download.Estimate`. `internal/steam/manifests.go`: `BuildID`, `FirstInstall`, `Paused`, `ReadManifest`.
- `internal/library`: `Game.Installing`, `Game.Paused`; the saved library's version goes up (5).
- `internal/ui/downloads.go`: the watching, the row and pane text, the blink.
- The list's delegate keeps a small `dlView` (the downloads, the pause marks, the clock), never the Model: a Model kept there keeps the list and its older delegate, one more copy at every blink.
- Only a download seen for the first time counts as started: one for an app the library never lists (a redistributable, a DLC) would read the library at every change of the log.

### Tests

- The reader on recorded log lines: a download from start to finish, a Workshop update ending with `scheduler finished` only, a pause and resume, two downloading at once, rates before the start ignored. The estimate's carry-on and its limits.
- The install file: `buildid`, first install, paused.
- UI: each phase's row and pane text, the row within 13 columns, the blink, a paused download from its install file, an ended download reading the library, a new one reading it too, one seen before not, the log read only when it changed.
- Real loop (lab, a Steam folder made in a temp folder): an update started while Quarter is open, going on, paused, resumed and ended; the library read only at those changes; nothing printed to the terminal.

## Moving fast through the list (Pablo, 2026-09-30: "lets go with c the moving dots"; the new loading "if the performance under the hood is better, lets do it")

From Pablo's video: holding down kept the last logo on screen (a picture is erased only when the next is drawn) while the pane showed other games, and every game passed was read and converted to a picture, then thrown away.

### What shows

- While a key is held (the selection moved twice within 150 ms), the band drops the last logo at once and shows three dots filling fixed places one by one (`██`, `██  ██`, `██  ██  ██`, none), 180 ms a step, in the accent colour.
- A single press never shows the dots or empties the band: the logo changes straight to the next.
- After landing, the dots show only when the logo is still on its way after 200 ms (a first download).
- Picked in the Scroll lab over A (the last logo stays), B (empty) and D (the game's name while moving).
- The wheel (fixed 2026-10-02, Pablo: "when scrolling with mouse wheel the 3 dots appear way to aften, i want it to look instant as if in navigating with arrow keys one click at a time"): a wheel click is a key press inside the mouse message, and both used to note the move, 0.5 ms apart, so every click looked like a held key and showed the dots. A move is now noted once (`moves`), so a wheel click is one step like one arrow press; only clicks spun faster than 150 ms apart count as held, as keys do. Test: `TestWheelStepIsOneStep`.

### Loading

- A logo is read only once the selection has stayed on the game 120 ms.
- Then the two games above and below are prepared from disk, so one step to a neighbour shows its logo at once. A neighbour with nothing drawable on disk is left until landed on (it is read, and downloaded if need be, then).
- At most 24 drawn logos are kept; the selected game's never goes.
- Measured on 255 of Pablo's logos: a conversion takes 15.5 ms and 16 MB of short-lived memory and keeps 13 KB. Holding past 100 games: about 100 conversions before (1.5 s of CPU, 1.6 GB churned), 6 now (the first press's game, the landed game, its 4 neighbours).

### Code and tests

- `internal/ui/scroll.go`: landing, holding, the dots, the neighbours, the 24 kept. Hooks in `Update` (`app.go`), `ensureLogo` and `logoView` (`logoart.go`), `wantPicture` (`picture.go`).
- Tests: no read while holding and the picture erased at the first quick step; a single step keeps the picture and shows no dots; landing reads the game and its neighbours once, a neighbour missing on disk is not read again; the dots for a late logo only after 200 ms, and they stop; the dots don't shift; at most 24 kept. Logo tests switch the landing wait off, as they do the store's.
- Real loop (lab, the real library, pictures on): holding past 100 games read 1 logo, landing 6; the dots in the band; nothing printed; Quarter's data untouched.

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

## Phase C: Performance

### C1. Measure (Pablo, 2026-09-30: "make the c1 careful professional plan first"; then "continue")

Measure only: nothing in the app changes in C1. The work runs in a copy of the code (`Documents\Quarter\Performance lab`); Steam is only read; every measurement runs 20 times and reports the typical run (median) and the worst, the first launch after a build separately. Results and raw numbers go in the lab; misses become the C3 list for Pablo to approve.

Machine: Intel Core Ultra 9 275HX, 31.4 GB, Windows 11 Home 26200, Windows Terminal 1.24.11911.0, Go 1.27.0, Balanced power plan.

Libraries: Pablo's (342 games, 12 installed, 304 sessions, past years) and a made one (5,000 games, about 20,000 sessions over 5 years, past years).

| # | What | Target |
|---|---|---|
| 1 | Launch to the intro's first frame, by step (start, saved files read in `main.go`, model, first frame) | under 100 ms (brief) |
| 2 | A key on the intro to a usable list | under 100 ms |
| 3 | First run: a full Steam scan | measured; runs behind the intro |
| 4 | Search keystroke | under 16 ms (brief) |
| 5 | The slowest frame of the 32 intros, at 80x24, 160x45, 240x67 | under 33 ms (goal 10 ms) |
| 6 | History keys: h, tab through the periods, page turns, the board | under 16 ms |
| 7 | Holding down with pictures on: time per key, output per second, pictures drawn, the landing's parts | under 16 ms per key |
| 8 | Idle 10 min: CPU and output (no game, a game running, a download) | under 1% of a core; no picture redrawn unchanged |
| 9 | Memory over a long run | flat |
| 10 | Disk: saved files and logos, now and projected | measured; nothing unbounded |
| 11 | No file, network or process work in the key and drawing code | none (brief) |
| 12 | Release size, 6 builds (Windows, macOS, Linux; amd64, arm64) | 8 to 12 MB (brief) |

Pablo's part: hold down 10 s in Windows Terminal and say whether a logo flickers, lags or is left behind (the Scroll lab video, 2026-09-30, covered this before the moving dots). Order: 11, 12, 4; 1, 2, 3; 5, 6, 7; 8, 9, 10. Out of C1: tmux and SSH.

### C1 results (2026-09-30)

| # | Result | Target | |
|---|---|---|---|
| 1 | Launch to first frame: 41 ms typical, 43 worst, 49 the first run after a build. Of that, 20 ms before `main` (14 ms package start-up, 12 of it `go-runewidth` v0.0.27 building its tables), under 5 ms Quarter's own work, 17 ms waiting for Bubble Tea's next frame | 100 ms | pass |
| 2 | Start screen key to list: 17 ms | 100 ms | pass |
| 3 | Full Steam scan: 7.3 ms (12 installed, 330 not) | measured | pass |
| 4 | Search keystroke: 2.5 ms (Pablo's), 3.4 to 4.6 ms (5,000); worst 10.5 | 16 ms | pass |
| 5 | Slowest intro frame: 3.9 ms (median 0.2 to 0.3); 80x24 shows the compact screen | 33 ms | pass |
| 6 | History keys: 3 to 6 ms (Pablo's); HIGH SCORES board at 5,000 games 11.7 typical, 21 worst | 16 ms | pass, board borderline at 5,000 |
| 7 | A held step with pictures on: 4.4 ms; the name picture 0.4 ms; Pablo: "holding is steady, nothing flickers" | 16 ms | pass |
| 8 | Idle 10 min: 1.52% of a core (no game), 1.46% (a game running), 0 bytes written, no picture redrawn; a download, 30 min: 3.3%, about 740 bytes a minute | 1% | miss |
| 9 | Memory: flat over 10,000 check rounds (0.87 to 0.91 MB) and over 30 min with a download (3 to 7 MB, no trend, 14 goroutines throughout) | flat | pass |
| 10 | Saved files: 1 ms to read (Pablo's); projected 5,000 games and 20,000 sessions: 6 MB, about 18 ms. The logos folder has no limit (90 KB a logo) | measured | note |
| 11 | No file, network or process work in the key and drawing code | none | pass |
| 12 | Release builds: 9.4 to 10.4 MB | 8 to 12 MB | pass |

Why idle misses: the running-game check and the download check (two messages each time), every 2 s, about 90 messages a minute, each rebuilding the whole screen with nothing new; during a download the arrow's blink adds 2 a second. And while a game updates, the library is still re-read every 30 s (60 scans in 30 min), though the download log now sees a download start and end.

C3 candidates, for Pablo to approve: one timer for the 2 s checks and no rebuild when nothing changed; the 30 s re-read only when the download log can't be read; `go-runewidth` v0.0.30 (its start-up work measured at 0 ms, about 12 ms off each launch); a size limit for the logos folder; the board at 5,000 games profiled (optional).

### C3 (Pablo, 2026-09-30: "go 1 to 3 now")

1. **One poll, and no redraw when nothing changed** (`internal/ui/poll.go`): one timer every 2 s asks Steam which game runs and looks at the download log, and brings both back in one message (it was two timers and three messages). Bubble Tea draws the screen after every message (`tea.go:888`), so after a round that found nothing new the screen drawn last is shown again; it is drawn anew after any other message, and at least every 30 s for the times on screen. A kept screen is tagged with the drawing it belongs to, so two model copies never share one.
2. **The 30 s reads while a game updates wait while Steam's download log is there** (`updating.go`): the log sees a download start and end, which read the library (`downloads.go`); the reads go on as the fallback if the log goes.
3. **`go-runewidth` v0.0.27 to v0.0.30** (indirect, through `charmbracelet/x/ansi`): its start-up work measured at 12 ms before, 0 ms after (`GODEBUG=inittrace=1`).

Measured again in the Performance lab, same tests:

| | Before | After | Target |
|---|---|---|---|
| Idle, no game, 10 min | 1.52% of a core | 0.87% | 1% |
| Idle, a game running, 10 min | 1.46% | 0.42% | 1% |
| A download, 10 min | 3.29% | 2.46% | 1% |
| Library reads during a download | 60 in 30 min | 0 in 10 min (the one at start) | |
| Messages a minute, idle | about 90 | 30 | |
| Launch to first frame | 41.3 ms (worst 43.0) | 33.7 ms (worst 35.0) | 100 ms |

Still over 1%: a download, from the arrow's blink, two full redraws a second. Not changed; options for Pablo.

Tests: a round asks the running game, and the log once watched; nothing to ask, no round; an unknown answer stops the asking; one round at a time; a quiet round keeps the screen, a game starting, a key or 30 s draw it again; two model copies never share a screen; with the log there the 30 s reads wait.

### C3, items 4 and 5 (Pablo, 2026-09-30: "4 lets do 3k and 5 do 5k")

4. **The logos folder keeps at most 3,000 downloads** (`config.LogoKeep`, about 270 MB at 90 KB a logo): past that, each download removes the least recently used. A saved logo read for drawing counts as used (its file time set to now). Only Quarter's folder; Steam's cache is never touched. `art.Sources.KeepAtMost`, set in `main.go`.
5. **Each screen drawn about five times cheaper** (profiled in the Performance lab): the quarter_ box was drawn again at every look at whether it fits, several times a screen, half of each drawing; it is drawn once now (`logo.go`). And in history, every look at the selected game sorted the whole ranking again; it is worked out once per state now (`history.go`), fresh before and after each message, so a new history shows at once.

| | Before | After |
|---|---|---|
| HIGH SCORES board at 5,000 games | 11.7 ms typical, 21 worst | 4.0 ms, 5.5 worst |
| A download's blink and its screen | 4.51 ms | 0.94 ms |
| A held step | 4.4 ms | 0.9 ms |
| A search keystroke (Pablo's) | 2.5 ms | 0.75 ms |

Tests: past the limit a download removes the least recently used, never the new one, and a read logo counts as used; the list of games without a logo stays. A new history heads the ranking at once.

Download, 10 min, after item 5 (the blink as before, twice a second): 2.46% of a core before, 1.29% after; about 600 bytes a minute to the terminal. Still over the 1% target; A (once a second) in the Blink lab for Pablo to compare.

The blink (Pablo, 2026-09-30, after the Blink lab: "the one second blink feels not wrong", and the pick left to the numbers): measured in the Blink lab, the same simulated download, 5 min each, after 30 s: twice a second 1.08% of a core, once a second 0.87%. Once a second it is: a download is now under the 1% target too.

### C4. Keeping the numbers (Pablo, 2026-09-30: "c4 go")

`TestSpeedTargets` (`internal/ui/bench_test.go`) now runs on 5,000 games, 20,000 sessions, store info for every game and four past years, and fails the build past its targets; each check is the quickest of five timings, each over a batch of runs for Windows' 0.5 ms clock. `TestIntroFrameSpeed` (`internal/intro`) times every frame of every intro at 240x67. Benchmarks for numbers: `go test ./internal/ui -bench .`.

| Check | Target | On Pablo's PC |
|---|---|---|
| Startup: saved files read (library, store info, sessions, past years), the model, the start screen's first frame | 100 ms | 31 ms |
| Search keystroke | 16 ms | 1.3 ms |
| A held step, pictures on | 16 ms | 0.8 ms |
| h, the ranking | 16 ms | 1.4 ms |
| h, the HIGH SCORES board | 16 ms | 3.4 ms |
| tab on the board | 16 ms | 3.8 ms |
| A download's blink and its screen | 16 ms | 0.9 ms |
| The slowest intro frame | 33 ms | 4.4 ms (Neon) |

Not guarded by a test: idle CPU (needs minutes; the Performance lab's `TestPerfIdle`), launch before `main` (`GODEBUG=inittrace=1`), release size (six builds). Measure them again before each release.

## Games installed or uninstalled while Quarter is open (Pablo, 2026-10-01: "ive just unisntalled a game and in quarter it didnt appear to be unistalled until i re open it"; "implement, with no test")

Found: DiRT Rally 2.0 uninstalled at 00:28:19. Steam logged `state changed : Uninstalled,` and `finished uninstall (No Error)`, removed `D:\SteamLibrary\steamapps\appmanifest_690790.acf`, and that folder's last-changed time became 00:28:19. Quarter re-read the library only on start, `r`, downloads starting and ending and a game stopping, so the game stayed listed.

- Each 2 s round (`poll.go`) now also looks at the last-changed time of each library's `steamapps` folder (Steam's own and every library with an installed game); a change reads the library at once. The first round only notes the times. With a read already running, the old times are kept so the next round looks again.
- It catches installs, uninstalls (Quarter's `u` or Steam's), games moved between drives, files deleted by hand and libraries added; it doesn't depend on the log's wording, and works the same on Linux and macOS. Only folder times are looked at: Steam stays read only.
- No new tests, at Pablo's word; the existing ones pass.

## Phase D: Release v0.1 (plan, 2026-10-01; not started)

From `PLAN.md` (Batch 8, Phase D) and the brief's license and Valve rules (`PROJECT_BRIEF.md`, "License and monetization"). Checked 2026-10-01: the repo has no README, LICENSE, privacy note, GoReleaser config or CI workflow yet; `main.version` is set at build time (`-ldflags "-X main.version=<tag>"`).

1. **D1. Pablo's decisions, first (hard to undo):** the license (MIT, or GPL-3.0 for the app with any Pro features in a separate private repo; the brief wants it decided before the first public commit), from a one-page comparison; whether `stronxyo/quarter` goes public with v0.1, and its final name; whether a Steam Deck is available for the smoke test; and, if the repo goes public, where Claude's memory backup goes (`.claude/memory/` and `.claude/global-CLAUDE.md` hold personal details: stay only while private, else a private repo or gist; `CLAUDE.md` at the root points to it).
   **Decided 2026-10-02:** MIT, everything free (Pablo: "maybe a donate button will do and anything is free. right? and let quarter be for the world?"), after research on similar tools (mostly MIT; cliamp MIT); the repo goes public with v0.1 as `quarter`; no Steam Deck (Linux ships marked untested); donations through Ko-fi and GitHub Sponsors (Uruguay is supported) in `.github/FUNDING.yml` and the README, his accounts to make; the memory backup moved to the private repo `stronxyo/quarter-notes`, and out of this branch's history (`git filter-branch` on the three commits from 3806601), before the repo goes public.
2. **D2. Documents:** README (what Quarter is, install, keys, terminals that draw pictures, "not affiliated with Valve", the Parzival story without book or film names); PRIVACY.md, which the Steam Web API terms need before a public release (what is saved only on the PC: library, store info, logos, sessions, past years; what is asked of the internet: store info, logos, past years only after `a`, the short-link service when a game's website is a short link, and the website or SteamDB only when the user opens them; nothing sent anywhere else; how to delete it all: `%APPDATA%\quarter`); LICENSE; CHANGELOG for v0.1.
   **Written 2026-10-02:** `README.md`, `PRIVACY.md`, `LICENSE` (MIT, Pablo Cardoso), `CHANGELOG.md` (0.1.0, unreleased) and `.github/FUNDING.yml` (Ko-fi and GitHub Sponsors as `stronxyo`). The README lists every install way planned in D4; in D4, any that doesn't happen (no account, name taken) comes out of it, and the Ko-fi and Sponsors pages must exist before the repo goes public. No screenshots for now (no VHS, Pablo 2026-10-01).
3. **D3. Checks before release:** measure again what the tests don't guard (idle CPU, launch before `main`, release size; see C4); a clean-terminal smoke test (a fresh data folder for the first run, Windows Terminal, the classic console, PowerShell, a narrow window), Pablo's real data untouched; Linux and macOS builds compile but can't be run here: shipped marked untested unless a machine turns up.
4. **D4. GoReleaser:** a config for six builds (Windows, macOS, Linux; amd64, arm64), stripped, the version stamped, zip for Windows and tar.gz for the rest, checksums; a local `--snapshot` dry run before anything is published. Pin the version after checking its current release.
   **One-line installs** (Pablo, 2026-10-01, after cliamp's README: "something like this would be nice"), a README section like cliamp's, all free with GoReleaser: `curl -fsSL https://raw.githubusercontent.com/stronxyo/quarter/main/install.sh | sh` (macOS, Linux) and `irm https://raw.githubusercontent.com/stronxyo/quarter/main/install.ps1 | iex` (Windows): two scripts in the repo fetching the right release file; `winget install quarter` (GoReleaser writes the manifest and PRs it to winget-pkgs); `brew install stronxyo/tap/quarter` (a `stronxyo/homebrew-tap` repo GoReleaser updates); `yay -S quarter-bin` (AUR; GoReleaser adds `-bin` for prebuilt packages; needs an AUR account and an SSH key); `nix run github:stronxyo/quarter` (a `flake.nix` in the repo); `go install github.com/stronxyo/quarter/cmd/quarter@latest`. Check first that `quarter` is free on winget, Homebrew and the AUR; Pablo creates the outside accounts. npm/npx left out: GoReleaser's npm publishing is Pro (paid) and needs Node. Sources: [GoReleaser: Winget](https://goreleaser.com/customization/publish/winget/), [Homebrew Casks](https://goreleaser.com/customization/publish/homebrew_casks/), [AUR](https://goreleaser.com/customization/publish/aur/), [v2.10 notes (npm, Pro)](https://goreleaser.com/blog/goreleaser-v2.10/).
5. **D5. CI (GitHub Actions), written by hand:** on every push and PR, the format check, vet on Windows and Linux, staticcheck, tests and the build; on a tag, GoReleaser. Actions pinned to exact versions. Check the speed tests on hosted runners, slower than Pablo's PC, without loosening their targets.
6. **D6. Pull request** `batch/08-release` to `main`: compare link, title and description, the post-merge script (Pablo's PR handoff rule; never `gh`). Pablo opens and merges.
7. **D7. Tag v0.1.0:** a git block for Pablo to push the tag; CI builds the release; check the six downloads on the release page.

Done when: v0.1.0 is tagged and its six downloads are on the release page.

## The donate link (Pablo, 2026-10-02: "the donate link can be inside the app too right? full right down corner...?"; his pick of four: "buy me a coffee ↗")

- `c buy me a coffee ↗` ends the key line, drawn like the other keys, flush with the window's right edge, at least 6 cells after the keys. `c` (Pablo, 2026-10-02: "make the buy me a coffe to have a quick letter input too, so for people that dont use mouse"; "go with c") or a click opens Ko-fi (`config.SupportURL`, `https://ko-fi.com/stronxyo`), lighting up under the pointer like other clickable text. On every key line (the list, history, search) and in the `?` view, not on the start screen. `c` works on the list and in history; typed into a search it is a letter. The `?` view lists it as `c coffee ↗`: its column's words must fit 12 cells for the view to fit 80 columns.
- Ko-fi (checked 2026-10-02 with a browser's user agent; Ko-fi answers 403 to plain requests): `ko-fi.com/stronxyo` and Pablo's ID link `ko-fi.com/M2B82831Z2` are the same page, "Support stronxyo". The app and `.github/FUNDING.yml` use the name, the README's Ko-fi button (Pablo's own snippet) the ID.
- Always on screen (Pablo, 2026-10-02: "the buy me a coffee must always be visible no matter the wide or if we are inside the ? help menu"; it first showed only when the keys left room): its room (22 cells: the link and a 3-cell gap) is kept first and the keys fit in the rest, giving way in the usual order (the arrows' hints, h history, the sort keys, w and b, then from the middle). At 80 columns the key line keeps enter, u, / and q next to it; every key is still in `?`. Too narrow for the link itself, it is cut. In the `?` view it has a line of its own at the bottom right.
- Tests: `TestSupportLink` (always at the right edge from 40 to 200 columns, the hover, the click opens Ko-fi); `TestSupportLinkInFullHelp` (its line in `?` at 80, 120, 200 columns, the view within the window); `TestShortHelpFits` (where each key gives way, the link always last); `TestSupportKey` (c on the list and in history, a letter in a search); the `?` view still fits 80 columns; 25 keys click in `?`, 9 on history's key line.

## The mouse (Pablo, 2026-10-01: "lets add it now v0.1, so make all the propper tests")

For people who don't live in a terminal. Designed in the Mouse lab (`Documents\Quarter\Mouse lab`, player `quarter-mouse`), Pablo's picks: double-click to play; the hover look "the same as the underline but without the line, just the color swap and also add the tini marker at the left but make it a small dot, maybe the squared dot"; three spaces between keys ("3 spaces look fine").

### What it does

- **Hover:** the text under the pointer turns amber, with `▪` (U+25AA, in both of Pablo's fonts) in the cell before it and a space between; not on the selected game or the active tab, which are lit already. Any key from the keyboard and a window resize clear it; after anything else it stays only while the same thing is still under the pointer (a filter shortening the list).
- **Clicks:** a game is selected; a second click on it within 400 ms plays it, or installs it when it isn't installed (as Enter). The same in the history ranking. Filter tabs and history periods; every key on the key line and on the full list after `?` (22 keys), each doing what its key does; the column names sort, again to flip; the game's logo opens the game in the Steam library (`steam://nav/games/details/<id>`; `s` stays the store page); the quarter_ box plays a random intro, then the start screen (an easter egg); any click leaves the start screen, like any key.
- **Right-click** a game: it is starred (as `f`).
- **Wheel:** moves the selection; over the tabs, turns them (down the next, up the one before).
- **HIGH SCORES calendar:** a day under the pointer says what was played then, under the pane: `Thu 1 Oct 2026 · 2.0h · Game 01, Game 04` (three games named, the rest counted).
- **While typing a search,** a click elsewhere ends the typing and keeps the results, then acts (no letters typed); a click on the search box takes the typing back; `esc clear` and the play key click as themselves. (Clicking the count to clear was tried and dropped: beside the search box the 80-column list has no room for the count.)
- **config.json `"mouse": "off"`** leaves the mouse to the terminal: plain text selection without Shift, the wheel as the terminal sends it.
- With the mouse on, copying text from the screen needs Shift + drag (Windows Terminal; most terminals the same).

### Code and cost

- `internal/ui/mouse.go`; `history.DayGames`; `calendarDays` and `sortColumns` shared by the drawing and the clicks; `hoverStyle`, `hoverDayStyle`, `markerHover` in `styles.go`; the key line's gap three spaces (`help.go`: `h history` now fits from 93 columns, page from 103, move from 113).
- What can be clicked is worked out from the drawn screen and kept until something other than the pointer moving happens (a hover changes looks, never places). Measured: a move along a row 0.16 ms, onto another row 0.71 ms (Pablo's library), 0.82 ms on 5,000 games. Idle unchanged: nothing runs while the mouse is still, and the hover doesn't animate.

### Tests

Targets match the screen at 150x40 and 90x30; double-click plays, a slow one doesn't, a game not installed installs; tabs, sort (and flip), the quarter_ box, clicks leaving the start screen; the logo hands Steam the library link; right-click stars; all 22 keys of the full list click like their keys at two widths, history's 6 periods and 8 keys; the search clicks; the wheel; the calendar day; the hover's look, width and space; the hover clearing; `"mouse": "off"`; `DayGames`. The speed check adds the pointer onto another game (16 ms). Real loop (lab, Pablo's library): the `▪` shows, a double-click asks Steam once, a right-click stars, nothing printed, his data untouched.

The logo click (Pablo, 2026-10-01: "instead of the store, we already have pressing s, it would be cool to go to that specific game on the steam library"): `steam.LibraryURI`, `steam://nav/games/details/<id>`, the form Valve's list of Steam links documents ([Valve Developer Community: Steam browser protocol](https://developer.valvesoftware.com/w/index.php?title=Steam_browser_protocol)). `steam://open/games/details/<id>` worked as well on Pablo's PC; the documented one was picked, to work on any OS (his rule: "use the most universal that can work on any machine any OS").

Games that aren't in the library (Pablo, 2026-10-01: "some games dont redirect me ... the library is opened on the last one that landed ok"; "see if we can open steam like a micro second later so the previous game is not seing"): Quarter lists every app with playtime or achievements on the PC (`library/scan.go`), which includes demos, playtests, free weekends and games no longer sold; Steam ignores a library link for those and shows the game it showed last. Two cases:

1. **No library artwork folder** (`steam.LibraryCacheDir`: `appcache\librarycache\<id>`, the folder Quarter's logos already come from, on every OS): Steam isn't opened; the message line says in red (`warn`) "Kentum Demo isn't in your Steam library (a demo, a playtest, or a game no longer sold)". On Pablo's PC, 70 of the 330 games not installed have no folder: 22 demos, 27 apps the store no longer has, and games like Deceive Inc.; it catches Kentum Demo and Deceive Inc., two of the three he named.
2. **Everything else:** the one link to the game's page. Steam draws the game it showed last, then the game: its own window, nothing a link can change.

Tried and dropped (Pablo, 2026-10-01: "the opening speed now is way worst and it goes from library main to the previous to the final game, a whole flashing show"): the library's home first (`steam://nav/games`), then the game's page 300 ms later. Looked for a sure way to know a game is in the library, and found none: the Web API's owned games need a key; the public profile's games list (XML and page) now asks to sign in; `localconfig.vdf` records the same for owned and not owned (Astro Burn, ASTRONEER: last played and playtime); Astro Burn keeps a full artwork folder though it isn't opened; Steam's license cache is encrypted and undocumented (against the most-universal rule). Known limit: a game like Astro Burn still opens on the last game. The installed-only alternative (the library only for installed games) was offered and not taken.

The check reads the disk, so it runs off the UI goroutine with the link (`openInLibrary`). Tests: the folder check (folder, file, missing); the click in both cases (the game's page once, or no Steam and the red message).

Steam's own word (Pablo, 2026-10-01: Astro Burn "opens the last opened game in library, this should not happen"; "run other games if you can to see if you can find all there is ... that can work on any device"; any check before "must at least clean the opening"): when Steam can't show a game, its interface logs `SteamUI: ERROR: Unrecognized request details <id>` in `logs\webhelper_js.txt`. Probed on Pablo's PC with consent, one library link per game for 24 games of every kind, the log read after each: installed 3, none refused; with an artwork folder 9, one refused (Astro Burn); without one 12, 11 refused (not Spacewar, 480, Valve's test app). Steam's artwork log (`steamui_librarycache.txt`) was measured too and dropped: it misses 1 of the 11 installed games and 61 of 261 with artwork.

- At start (once the Steam folder is known), Quarter reads that log's refusals off the UI goroutine (`steam.LibraryRefusals`, 13 ms for the 6 MB log, 17 games on Pablo's PC): a game Steam refused in the last 30 days isn't opened; the red message shows at once. An installed game is in the library whatever the log says; after 30 days a refused game is tried again (bought since, maybe).
- After a link, the log is watched 3 s (every 250 ms, `steam.RefusedSince`); a refusal brings the red message and the game is remembered. Only a game never tried before, with artwork, still shows the last game once.
- The opening itself is unchanged: one link, nothing before it (Pablo: the single link is better than either form with anything before it). Undocumented log: missing or worded otherwise, nothing is found and Quarter behaves as without it.
- Tests: the log reader (latest time per game, a 2 MB line, Windows line ends, the literal `<id>`, no log; the same second counts, earlier doesn't); the click learning a refusal and the next click not opening Steam; refusals read at start, once, installed and 30-days-old ones not blocked. Against Pablo's real log: 17 blocked, Astro Burn among them, Rocket League (installed) not.

## Attract mode on the start screen (Pablo, 2026-10-02: "lets build this first ... make the shine sweep to apear randomly between 10 secs and 20 secs. then the random intro vid again in random time between 45secs and 60 secs"; no lab, all parts already seen)

- As "insert quarter" waits, the Light sweep intro's band of light crosses the logo (0.7 s, colour only, insert quarter keeps blinking) every 10 to 20 s, counted from the last shine's end (`intro.Screen.Shine`, sharing `shine` with the Light sweep intro).
- Every 45 to 60 s, counted from the screen coming to rest, an intro plays again: random and never the one before, or the named one; none with `"intro": "off"` (the shine still shows). Once it ends, both waits start again.
- A key during a replay starts (the first intro's key still only skips it). Too small a window (the compact screen): neither.
- Both are timed by the blink tick that already runs; only the shine's 0.7 s and a replay draw at 30 fps.
- Tests (`splash_test.go`): the shine's times, colour only, ending at rest, no replay when off; three replays in a row, each different from the one before, waits reset after each, a key starts; the small window quiet.

## Publisher, website and SteamDB (Pablo, 2026-10-02, after the Details lab: "i like publisher and website on the details, just separate them by a space with the above intel. make add the w website and b steamDB to the keys below.")

- The pane: after the table (Library or Status), a blank line, then `Publisher` (the store's first) and `Website` (shown as its name with an up-right arrow, `rocketleague.com ↗`; a cut name keeps the arrow). No terminal link codes (OSC 8): Windows Terminal underlines them with white dots (Pablo, 2026-10-02: "remove the white doted underline in the websites link and add an arrow pointing up right at the end"). The arrow, U+2197, is in JetBrains Mono; on Cascadia and Consolas Windows draws it from Segoe UI Symbol, like the favourite star. Compared with → (in every font) on the three fonts (Details lab, screens/arrow-fonts-compare.png); Pablo kept ↗ (2026-10-02: "leave the upright"). → stays the fallback if the classic console shows a box in D3. Only what the store gave; neither, no block. No SteamDB row (the lab's other layouts and the SteamDB row were not chosen).
- Keys: `w` opens the website, `b` the game's SteamDB page (`https://steamdb.info/app/<id>/`, an unofficial site, only linked), for any selected game, running too. Without a website (none listed, not on the store, store info not here yet) w leaves the key bar and does nothing, like i on an installed game (Pablo, 2026-10-02: "if there is no web site ... remove the button ability"); b always stays, since SteamDB has a page for every Steam app and can't be checked first (it blocks automated requests). Both stay listed in `?`, the full reference. A failed open says "Could not open the browser".
- The key bar: `w website   b steamdb` after the game's keys; on narrow windows they give way after h history and the sort keys (under 92 columns), always in `?`.
- Safety: the website is kept only as a full http or https link (`webapi.website`), so nothing else from the network is handed to the OS to open.
- Short links (Pablo, 2026-10-02: "for example in once human put https://www.oncehuman.game/ not t.co/... use the official site domain names"): when the store's website is on a link-shortening service (t.co, bit.ly, goo.gl, tinyurl and others, `webapi.shortHosts`), Quarter asks that service where it leads (HEAD, up to 5 short links in a row) and stops at the first address that isn't one, so the game's own site is never visited. That address is kept. One that can't be followed (offline, dead, a loop, not a web page) leaves no website: a short link is never shown. Measured 2026-10-02: Once Human gives `https://t.co/gRX2BR4tj9`, one redirect to `https://www.oncehuman.game/`; of 245 websites in Pablo's library, only t.co links were short. PRIVACY.md (D2) must say the short-link services are asked.
- The store cache goes to version 4 (3: the website; 4: short links followed): the saved details are asked for again (the selected game first, the rest in the background at the usual pace).
- Tests: the website parsed and filtered (other schemes, bare words); the block after a blank line, the name and its arrow (kept when the name is cut), no link codes, no block without data; w and b open the right links, a running game too; without a website w is off the key bar and does nothing, b stays; the browser named on a failed open; the website clicks like w, lights up, and wins over the row under the pop-up, at 150 and 90 columns; the key bar's widths; 24 keys click in `?`. - The mouse (Pablo, 2026-10-02: "and the mouse click doesnt work?"): Quarter has the mouse, so the terminal's own link click never reaches the website. The website's name in the pane (beside the list or in the pop-up) is one of Quarter's click targets: it lights up under the pointer and a click does what w does, in every terminal. It comes first among the targets, so it wins over the game row under the pop-up. With "mouse": "off", w is the way.
- For D3: check the arrow draws in the classic console.

## The classic Windows console (D3, Pablo, 2026-10-02: after the smoke test's boxes and the Console lab: "we go with c")

The classic console (conhost) draws a character missing from its font as a box: it borrows nothing from other fonts, as Windows Terminal does. Its default fonts, Consolas and the Cascadia Mono that ships with Windows, lack the arrow ↗, the stars ✦ ✧, ▶ and ▹, the quarter and eighth blocks. Measured on the font files with fontTools.

- **Who meets it (research, 2026-10-02):** Windows 11 opens console programs in Windows Terminal by default since 22H2 (October 2022); Windows 10 still opens the classic console. Steam's survey, August 2026: Windows 11 about 71%, Windows 10 about 24% and falling (Windows 10 support ended October 2025, free updates to October 2027). Sources in the session: Neowin, IBM, BleepingComputer, microsoft/terminal #13153, Tech Insider, The Register, Microsoft.
- **What it gets:** no game logos (half blocks alone couldn't draw them cleanly: variants A and B tried in the Console lab, Pablo picked C), and stand-ins both fonts have: → for ↗, ♦ ◊ for ✦ ✧, ► for ▶, › for ▹, ░▒▓█ for the spark bars, ▌ for the bar ends, ♦ for ◆ in the Card flip intro (`internal/ui/glyphs.go`, `intro.BasicGlyphs`). `o` there says logos show in Windows Terminal. Everything else as anywhere. The logo renderer is untouched.
- **How it is found:** not by environment variable (a console opened from Windows Terminal inherits its `WT_SESSION`; that hid the lab's first try). Quarter asks Windows: the console window (`GetConsoleWindow`) of the classic console has the class `ConsoleWindowClass` and is visible; Windows Terminal, VS Code and others host Quarter through a pseudoconsole with a hidden window of another class. Proved with a probe in a classic console opened from Windows Terminal (classic) and in Windows Terminal (not). `config.json`'s `"glyphs"`: `""` found out, `"basic"`, `"full"`. Lucida Console lacks even the frames and isn't covered.
- **The goodbye** in the classic console showed raw colour codes: printed after Bubble Tea gives the console its old mode back. Now virtual terminal processing is switched on for it and the mode put back; where that can't be done, plain text (`colourOutput`, `printGoodbye`).
- **Tests:** `TestBasicGlyphsEverywhere` drives every screen (list with every kind of row, running, downloading, favourites, details, pop-up, `?`, every hover, history, board, past years question, search, Steam not found, no games, a damaged file, a library offline, the store offline) at 60x20, 80x24, 120x40, 200x60, every frame of all 32 intros, the shine and all 11 goodbyes at two sizes, and fails on any character outside the list checked in both fonts; it caught ↗ when put back on purpose. `TestClassicConsoleNoLogos` (no logo drawn, no picture, none read or downloaded, `o`'s message; another terminal still shows it; caught the rule removed on purpose). `TestGlyphSetsMatch`, `TestBasicGlyphsSetting`, `TestPrintGoodbye`. Real program loop in the Release lab on a copy of Pablo's data at three sizes: 580 KB of terminal output, nothing outside the fonts (game text “ ” — ö checked in both), no picture data, no log lines.

## D3 results (2026-10-02)

| Check | Result | Target |
|---|---|---|
| Release size, 6 builds | 9.6 to 10.7 MB | 8 to 12 MB |
| macOS, Linux | build, vet and staticcheck clean (not run on a real machine) | compile |
| Start-up before `main` | 2.9 ms typical, 6.7 worst (14 ms at C1) | small |
| Startup to first frame; every key | 31 ms; under 4 ms | 100 ms; 16 ms |
| Idle on the list, 10 min | 0.16% of a core, nothing written, memory flat | 1% |
| Idle on the start screen, 10 min, attract mode | 1.72% before, 0.77% after the fix below (same blinks, shine and replays) | 1% |
| Smoke test (Pablo): Windows Terminal, classic console, PowerShell, narrow; fresh data folders | the classic console's boxes and goodbye codes found and fixed (above), then the console run again: "looks good"; the real data folder untouched by the smoke runs | |

The start screen's cost, profiled (3 min): most of it Go waking for each animation frame and Bubble Tea turning frames into output; Quarter's own share was laying out the hidden list on every frame (the key line most of all) and laying out and drawing the start screen again for every blink. Now the list waits until the start screen goes (the key that leaves it lays it out), and the start screen's layout and its two resting pictures are kept while the window and check stay the same (`startMemo`). Tests: `TestStartScreenKept` (kept pictures equal fresh ones through blinks, a resize, a new scan), `TestListLaidOutAfterStart`.

## D4 results (2026-10-02, Pablo: "go"; the AUR name "go for quarter-tui-bin")

- **Names:** `quarter` is free on winget (no package, no moniker), Homebrew (not in Homebrew's catalogue; ours is in Pablo's tap) and nixpkgs. On the AUR `quarter` is Coin3D's Qt library, so the package is `quarter-tui-bin` (GoReleaser's AUR packages end in `-bin`; a from-source `quarter-tui` could come later). It provides and conflicts with `quarter-tui` only: GoReleaser's default, conflicting with the project name, would have blocked Coin3D's `quarter` (seen in the dry run, fixed). The command is `quarter` everywhere.
- **`.goreleaser.yaml`** (GoReleaser v2.18.2, the latest, 2026-09-17): six builds, `CGO_ENABLED=0`, `-trimpath -s -w`, the tag stamped into `main.version`; zip for Windows (winget needs zip), tar.gz elsewhere, each with README, LICENSE, CHANGELOG and PRIVACY; `checksums.txt` (SHA-256); the GitHub release; a Homebrew cask in `stronxyo/homebrew-tap` (the program isn't signed by Apple, a paid account, so the cask clears macOS's quarantine flag after install, as GoReleaser's docs suggest); a winget manifest `stronxyo.Quarter` as a portable zip with the command `quarter`, sent as a pull request to `microsoft/winget-pkgs` from Pablo's fork; the AUR package over SSH.
- **Dry run** (`goreleaser release --snapshot --clean`, publishes nothing): six archives of 3.7 to 4.2 MB, every checksum verified, the Windows build runs, the cask, the winget manifests and the PKGBUILD read through.
- **Install scripts** (`install.sh`, `install.ps1`): find the latest release (the release page's redirect, or GitHub's API in PowerShell), download this system's archive and `checksums.txt`, refuse anything that doesn't match, install to `~/.local/bin` or `%LOCALAPPDATA%\Programs\quarter` with no admin rights, and put it on the PATH (PowerShell: the user's PATH; sh: says which line to add). The PowerShell script runs in a block of its own, so nothing stays in the user's window. Settings for a fixed version, the folder and a test server. Tested against the dry run on a local server: Windows PowerShell 5.1 and PowerShell 7 install and add the PATH (Pablo's own PATH restored after); a tampered download refused by both scripts, nothing installed, PATH untouched, no temporary files left; the sh script installs the Linux build (from Git Bash, the program itself not run: no Linux here) and on Windows points to the PowerShell line. `.gitattributes` keeps `.sh` files in Unix line endings.
- **Nix moved to after v0.1:** a flake needs a hash of the dependencies only Nix can compute, and there is no Nix on this PC; the README's `nix run` line is out until CI can build it on Linux.
- **For D5/D7, Pablo's part:** a public repo `stronxyo/homebrew-tap`; a fork of `microsoft/winget-pkgs`; one classic GitHub token with `public_repo` (a fine-grained token can't open pull requests on Microsoft's repo), stored as the secrets `HOMEBREW_TAP_TOKEN` and `WINGET_TOKEN`; an AUR account with an SSH key, the private key as the secret `AUR_KEY`. Microsoft reviews a first winget package by hand, so `winget install quarter` works some days after the release.

## Steam starts quietly (Pablo, 2026-10-08: "lets use this steam -silent -applaunch <id>"; "go")

Before v0.1, from `specs/STEAM_QUIET_START.md` (research and Pablo's live tries there). No lab for the look (Pablo: "we dont need lab for this"); the real program loop was checked in `Documents/Quarter/Steam start lab`.

### What it does (Windows)

1. **Quarter opens with Steam closed:** it starts `steam.exe -silent` (Valve: "Silent startup mode (tray mode only)"), off the UI goroutine; Steam goes to the tray, no window. Steam already running: nothing.
2. **Play with Steam ready:** the `steam://rungameid/<id>` link, as before, "Launching X".
3. **Play with Steam closed, or while the start Quarter made is still signing in:** `steam.exe -silent -applaunch <id>`, which Steam holds until it has signed in (tried by Pablo). The message line under the pane says "Opening Steam" with three moving dots, filling like the logo band's but the hover's small square `▪` (the band's blocks looked too big beside text; Pablo: "the small square is good"; in Consolas, Cascadia Mono and JetBrains Mono), until the game runs (`RunningAppID`), then goes; the row says running as before. Play again while it waits: nothing.
4. **At 60 s with no game** (Pablo: "90 is too much, make it 60"): Steam not running ends the wait, light red: "Steam didn't start. Press enter to try again." Steam still signing in (`ActiveUser` 0): no resend, the game is already held; the message says "Steam is still signing in". Steam signed in without the game: `-applaunch` once more, "Still opening Steam". Safe: Steam sets `RunningAppID` 0.55 s before the game's window shows, so a game already starting isn't sent twice.
5. **At 120 s still no game:** the wait ends, light red: "Steam hasn't signed in; open it from the tray.", or "Steam is open but didn't start X; it may be updating. Click the logo to see Steam, or press enter to try again."
6. **Install, uninstall, the store page, the logo click:** unchanged, Steam's window shows (Pablo: "yes").
7. **Quitting Quarter:** Steam stays in the tray. Steam is started detached from the terminal.
8. **`"steam_start": "normal"`** in config.json: as before, no start at opening, play by the link (Pablo: "agree" to a setting, default quiet). Only in config.json, explained in the README's settings section, no key (Pablo: "lets keep the other on the config but explained on the readme").
9. **Linux and macOS:** as before (the fallback): no one can try `-silent` there for Quarter yet, and on macOS `open -a Steam --args` drops the arguments once Steam runs.

### How Steam is read

- **Running:** `HKCU\Software\Valve\Steam\ActiveProcess\pid` is a live process whose program is `steam.exe`. Steam leaves the pid there after quitting (seen 2026-10-07: a stale 29952, later 2568, with no Steam running), so the pid alone isn't enough.
- **Signed in:** `ActiveProcess\ActiveUser` not 0. Measured 2026-10-08 (a read-only watch every 0.5 s while Pablo ran the commands): Steam's pid live at once; `ActiveUser` still last run's account for 1.6 s, then 0 while signing in, then the account at 26.6 s; `RunningAppID` the game at 30.9 s, the game's window at 31.5 s; `RunningAppID` back to 0 within 3 s of quitting. So a start Quarter made counts as finished only once Steam was seen signing in and then signed in, or signed in 10 s after the start; a start not seen finished in 120 s is no longer waited on.
- **steam.exe:** `--steam-path`'s folder when given, else the registry's `SteamExe`, else the folder Quarter finds Steam in.
- Steam's state is read only while Quarter waits on its own start or on a game (one registry read and one process look per 2 s round); idle rounds cost what they did.

### Code and tests

`internal/steam/client.go`, `client_windows.go`, `client_other.go`; `internal/ui/steamstart.go`, with `app.go`, `actions.go`, `poll.go`, `running.go`, `settings.go`; `config.Settings.SteamStart`; `cmd/quarter/main.go`. Tests: `TestIsSteamImage`, `TestLiveSteam` (a live non-Steam pid and a finished pid aren't Steam), `TestSteamStartsAtOpen`, `TestPlayPicksQuietOrLink`, `TestBootEndsWhenSignedIn`, `TestLaunchWaitEnds`, `TestLaunchChecks`, `TestPollAsksSteamOnlyWhileWaiting`, `TestSteamStartSettingKept`; each checked to fail with its part broken (no single resend, no stale-account grace, the wait not ending when the game runs). The real loop (lab, fake Steam, size sent twice, log to a file): Steam started once at opening, one quiet play for two presses, "Opening Steam" on screen, gone once the game runs, nothing but the screen in the terminal; it fails with the start or the wait's end taken out. Every test, `TestSpeedTargets` unchanged, vet and staticcheck pass; Linux and macOS build.

**Not tested:** Steam signed out (Steam shows its sign-in window: it needs the user, so the start isn't quiet then).

## D5 results (2026-10-10, after Pablo made the tap)

Two workflows in `.github/workflows/`, written by hand:
- **`ci.yml`**, on every push and pull request: `gofmt -l` and staticcheck 2026.2.1 on Linux (gofmt only there: Windows checks files out with CRLF, which gofmt reports on every file); then `go vet`, `go test ./...` (speed targets included, unchanged) and `go build ./cmd/quarter` on Windows, Linux and macOS. macOS was added beyond the plan: it's the only place the macOS build is tested at all.
- **`release.yml`**, on a `v*` tag: the tests once more, then GoReleaser v2.18.2 with `contents: write` and the secrets `HOMEBREW_TAP_TOKEN`, `WINGET_TOKEN`, `AUR_KEY`.
- **Pinned** to full commit hashes, as GitHub's guide on securing Actions advises, the release beside each: `actions/checkout` v7.0.1 (3d3c42e), `actions/setup-go` v7.0.0 (b7ad1da), `goreleaser/goreleaser-action` v7.2.3 (f06c13b); latest releases on 2026-10-10 from each repo's releases page. Go comes from `go.mod`.
- **The AUR skips itself** until the `AUR_KEY` secret exists: `skip_upload` is a template in GoReleaser (expanded in `internal/pipe/aur/aur.go` of v2.18.2, "true" skips the upload), set to `true` unless `isEnvSet "AUR_KEY"`; `private_key` reads it with `envOrDefault`. The package file is still built into `dist/aur/`.
- **Checked here:** gofmt, `go vet` for Windows, Linux and macOS, staticcheck 2026.2.1, every test, `goreleaser check`, and a snapshot (six archives, checksums, cask, winget and AUR files) all pass.
- **First run on GitHub (ci #1, 24dc155):** the format check and staticcheck passed; tests failed on all three systems, three causes:
  - `TestIsSteamImage` (Linux, macOS): `isSteamImage` split a Windows path with `filepath`, which on Linux and macOS splits only on `/`. It now splits on either slash on every system.
  - `TestRealCase` (Windows): the runner's temp folder is a short 8.3 name (`RUNNER~1`), which `realCase` rightly leaves alone (its result must match the path it was given), so the test's own expectation was wrong there. The test now works under the long name (`GetLongPathNameW`); reproduced here with a short-named temp folder: the old test fails, the new passes.
  - `TestSpeedTargets` (Linux): startup 137 ms against 100 ms. The runner ran every check 3.5 to 4.3 times slower than Pablo's PC, and startup had the least room. The target stayed; the start got faster instead, for everyone: the list's sort lowered every name at each comparison (now once per game), and the four saved files are read at the same time in `runUI` (the test's `startup` does the same). Startup here 33 ms to about 10 to 14 ms; held to 2 cores (GOMAXPROCS=2, like a hosted runner) about 16 ms. Every other speed check unchanged.
- **Checked again here:** gofmt, vet and staticcheck for Windows, Linux and macOS, every test.
- **Second run on GitHub (ci #2, c6916cf, 2026-10-10): passed** on Windows, Linux and macOS, the speed targets included, in 2 min 59 s. (Passing tests print no times without `-v`, so the runners' own numbers aren't shown.)
