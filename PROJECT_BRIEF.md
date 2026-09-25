# Project Brief: Quarter

## Name

**Quarter** (binary: `quarter`). Named after the quarter Parzival wins in *Ready Player One* for his perfect game of Pac-Man, an extra life that saves him in the final battle. It's also what you drop into an arcade machine to start playing, which is exactly what this app does.

Theme touches stay light: an "INSERT COIN" splash on first run, `1UP` as the play hint, "GAME OVER" on quit. Nothing more than that.

## One-liner

A fast, good-looking terminal app to browse, search and launch your Steam library. Think "cliamp, but for your games." Fuzzy search, playtime, achievements, each game's logo redrawn as retro text art, press Enter to play.

## Why

- Existing Steam TUIs (steam-tui, Aurelia TUI) are Linux-first and try to replace the Steam client, which is fragile and risks the user's account.
- Nothing polished exists for Windows that sits on top of the real Steam client.
- Terminal tools spread through screenshots and GIFs. A beautiful, classic-looking TUI (block-art logos, no images) is the marketing.

## Core principles

1. **Never replace Steam.** The official Steam client handles downloads, updates, DRM, anti-cheat, overlay and cloud saves. We only read data and hand off actions via `steam://` URIs.
2. **Never handle Steam credentials.** No username or password, ever. Optional Web API key only.
3. **Instant.** Opens in well under a second after the first run. Everything cached locally, network fetches are lazy and in the background.
4. **Keyboard first.** Every action has a key. Mouse is optional.
5. **Windows first**, cross-platform after (Windows Terminal is the primary target).

## Performance targets

Speed comes from architecture, not language: the bottlenecks are disk, network and terminal drawing, never CPU. These are hard targets, measured, not aspirations.

- **Startup:** under 100 ms from launch to an interactive list, served from cache.
- **Never block the UI:** all disk and network work runs in background commands and arrives as messages.
- **Search:** no dropped keystrokes on a 5,000-game library.
- **Binary size:** small, built with `-ldflags "-s -w"` (expect roughly 8 to 12 MB).
- **Profile before optimizing:** use `pprof` to find real hot spots. Profile-guided optimization (PGO) only if profiling shows it's worth it.
- A startup benchmark lives in the repo so regressions are caught.

## Platforms

Go cross-compiles everything from one machine. Release targets:

| OS | Architectures |
|---|---|
| Windows | amd64, arm64 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Linux | amd64, arm64 (includes **Steam Deck**, a natural audience) |

- Release via **GoReleaser**: GitHub Releases, plus Scoop and winget (Windows) and Homebrew (macOS/Linux).
- Requirements: a terminal and an installed Steam client. Nothing else.

## Tech stack

- **Language:** Go
- **TUI:** Bubbletea + Lip Gloss (same stack as cliamp)
- **Output:** single binary, no runtime dependencies
- **Config/cache location:** `%APPDATA%\quarter\` on Windows, `~/.config/quarter/` elsewhere
- **Libraries:** `github.com/sahilm/fuzzy` for search. A text-VDF parser (existing Go package or a small hand-written one). The binary `appinfo.vdf` needs its own reader, which is deferred past v0.1.

## Data sources

Rule of thumb: **local first.** Steam keeps most of what we need on disk. The Web API is only for what can't be read locally.

### Local (no login, no network)
- Steam install path: registry `HKCU\Software\Valve\Steam`, value `SteamPath`
- Library folders: `<SteamPath>\steamapps\libraryfolders.vdf` (lists every library across drives)
- Installed games: `appmanifest_*.acf` in each library's `steamapps` folder (contains `appid`, `name`, install state, size)
- **SteamID64:** `<SteamPath>\config\loginusers.vdf` lists the accounts logged in on this machine; each block's key is the SteamID64. Pick the account with `MostRecent` = 1 if present, otherwise the highest `Timestamp` (recent Steam clients stopped writing `MostRecent`). The setup wizard should never ask for it.
- **Playtime and last played:** `<SteamPath>\userdata\<accountid>\config\localconfig.vdf`, per app. No key needed. (`accountid` = SteamID64 − 76561197960265728.)
- **Game logos:** Steam's own cache at `<SteamPath>\appcache\librarycache\<appid>\logo.png`, or one folder down in a hashed subfolder. Fall back to the CDN when missing.
- **Non-Steam shortcuts:** `userdata\<accountid>\config\shortcuts.vdf` (binary VDF). Later phase, but a cheap step toward the v1.0 unified library.
- **Steam favourites (optional import):** `userdata\<accountid>\config\cloudstorage\cloud-storage-namespace-1.json`. Later phase.
- **Filter out non-games:** v0.1 uses a hardcoded denylist of appids (Steamworks Common Redistributables `228980`, Proton versions, Steam Linux Runtime, etc.). Later, read the app type from `appcache\appinfo.vdf` (binary) instead.

### Steam Web API (optional, needs a free user API key)
- `IPlayerService/GetOwnedGames` with `include_appinfo=true`: owned-but-not-installed games (local data only covers installed ones)
- `ISteamUserStats/GetPlayerAchievements`: per-game achievement progress. One call per game, so fetch lazily for the selected game.
- `ISteamNews/GetNewsForApp`: patch notes and news (later phase)
- **Private profiles:** if the profile's game details are private, `GetOwnedGames` comes back empty; if only playtime is private, every playtime is 0. Detect both and explain them in the UI rather than silently showing nothing.
- **Key storage:** v0.1 stores the key in plain text in `config.json`, and the wizard says so. Windows Credential Manager is a later option.

### Store metadata
- Store app details endpoint (`store.steampowered.com/api/appdetails?appids=<id>`): description, genres, developer, release date, screenshots
- Unofficial and rate-limited (roughly 200 requests per 5 minutes): fetch for the selected game first, backfill slowly in the background, cache aggressively, never block the UI on it

### Game logos (decided in Batch 4: no images, text art only)
- Order of preference: Steam's local `librarycache` (direct, then hashed subfolder), then our disk cache, then Steam's CDN `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/<appid>/logo.png` (then cache it), then the game's name in a block font
- Drawn as eighth-block text art with the QUARTER gradient; details in `specs/BATCH_04_SPEC.md`

## Actions (via Steam URIs)

| Action | URI |
|---|---|
| Play | `steam://rungameid/<appid>` |
| Install | `steam://install/<appid>` |
| Store page | `steam://store/<appid>` |

If Steam is closed, these open it first. That is expected behavior.

## MVP (v0.1) features

### Library pane
- All installed games from all library folders
- If API key is set: also owned-but-not-installed games, marked differently
- Fuzzy search as you type (`/` to focus)
- Sort: name, playtime, last played
- Filters: All, Installed, Recently played, Favourites
- Status markers: installed, not installed, favourite

### Detail pane (selected game)
- Game logo: the `logo.png` silhouette redrawn in eighth-block characters with the QUARTER gradient (sixth blocks as a fallback setting). No images.
- Developer, release year, genres
- Playtime and last played
- Achievement progress bar
- Short description

### Keybindings (initial)
| Key | Action |
|---|---|
| `↑/↓` or `j/k` | Move |
| `/` | Search |
| `Enter` | Play |
| `i` | Install |
| `s` | Open store page |
| `f` | Toggle favourite |
| `Tab` | Cycle filter |
| `o` | Cycle sort |
| `r` | Refresh library |
| `?` | Help |
| `q` | Quit |

### First-run setup
- Auto-detect Steam, scan libraries, show the list immediately
- Optional setup wizard (like cliamp's) to paste a Web API key, with a link to where to get it. SteamID64 is auto-detected, never asked for.
- Works fully without the key: installed games with playtime, last played and their logos. The key adds uninstalled owned games and achievements.

## Layout sketch

```
┌ Library ─────────────── 142 games ┐┌ Hades II ────────────────────────┐
│ / hades                           ││ [game logo in block art]         │
│                                   ││                                  │
│ ● Hades                    48h    ││ Supergiant Games · 2025          │
│ ▶ Hades II                 12h    ││ Action Roguelike, Mythology      │
│ ○ Hadean Lands              -     ││                                  │
│                                   ││ Played 12h · Last: 3 days ago    │
│                                   ││ Achievements ███████░░░ 28/49    │
│                                   ││                                  │
│                                   ││ Battle beyond the Underworld...  │
├───────────────────────────────────┴┴──────────────────────────────────┤
│ [All] Installed  Recent  Favs    ⏎ play  i install  f fav  ? help    │
└───────────────────────────────────────────────────────────────────────┘
```

Responsive: on narrow terminals, collapse to list only and open details as an overlay.

## Caching

- Library index, metadata and API responses cached as JSON in the app data folder
- Images cached as files keyed by appid
- Show cached data instantly on launch, refresh in the background, update the UI when new data arrives
- Metadata refresh window: days, not minutes (it rarely changes). Playtime/achievements: refresh on launch.

## Risks

- **Eighth-block characters are new (Unicode 16, 2024).** Older fonts may not draw them. Mitigation: sixth blocks as a fallback setting. (Sixel images were prototyped in Batch 4 and rejected on looks.)
- **Undocumented local formats.** `localconfig.vdf`, `librarycache` layout and `appinfo.vdf` can change with Steam updates. Parse defensively: a failed read degrades a feature, it never crashes the app.
- **Store API rate limits.** See Store metadata above.

## Roadmap after MVP

- **v0.2:** tags and custom collections, backlog status (playing, finished, dropped), themes
- **v0.3:** news/patch notes per game, "random game from backlog", stats screen (total hours, most played, never played)
- **v1.0:** Epic Games and GOG detection and launching (unified terminal game library)

## License and monetization

Monetization is a later option, not a v0.1 goal, but these decisions are made **before the first public commit** because they're hard to undo.

**Possible models** (most to least common for terminal tools):
1. Open source + donations (GitHub Sponsors, Ko-fi), the cliamp model
2. Open core: free app, paid "Pro" features (e.g. Epic/GOG support, stats, themes, collection sync)
3. Pay-what-you-want builds on itch.io or Gumroad, with the source still free
4. Indirect: reputation, freelance work, jobs

**License:** undecided. MIT lets anyone fork and sell it. GPL for the open core, with any Pro features in a separate private repo, keeps more options open. Decide before going public.

**Rules to follow:**
- **Valve trademarks:** no "Steam" in the name or logo, and nothing that implies Valve endorses it.
- **Steam Web API:** follow its Terms of Use: 100,000 calls/day; no implied Valve endorsement; Steam Data shown "as is" with disclaimers; tell users what Steam Data is stored (needs a privacy policy before any public release). Charging is not prohibited. Re-read them before charging money.
- **Store `appdetails`:** undocumented and unofficial. Keep it an optional enhancement. A paid feature must never depend on it.
- **Game logos** belong to Valve and publishers: read them from the user's own cache or the CDN and redraw them at runtime; never bundle or rehost them.
- **Ready Player One:** "Quarter" is a common word and fine to use. The Parzival story can go in the README, but no book or film names, logos or artwork in anything sold.

Not legal advice. Get the Valve terms properly checked before launching anything paid.

## Out of scope

- Downloading/updating games ourselves (Steam does it)
- Logging into Steam, friends, chat
- Physical collection tracking
- Any GUI

## How to start (for Claude Code)

1. Read this brief fully.
2. Propose a project structure and a short implementation plan for v0.1 before writing code.
3. Build in this order: Steam detection and library scan (including local playtime), basic list UI with search, launching, then detail pane, then API key support, then game logos (the Batch 4 spike settled on block-art logos, no images). See `PLAN.md`.
4. Keep each step runnable and testable on Windows.
