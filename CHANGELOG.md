# Changelog

All notable changes to Quarter. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.2.1] - 2026-10-19

### Changed

- The macOS and Linux install script makes typing `quarter` work: when `~/.local/bin` isn't on your PATH, it adds it in your shell's settings file (`~/.zshrc`, `~/.bashrc` or `~/.bash_profile`, fish's `conf.d`, or `~/.profile`) and says to open a new terminal. `QUARTER_NO_MODIFY_PATH=1` leaves your settings alone.
- The Windows install script finds the latest release without GitHub's API, so a shared connection can't run into the API's limit. Both scripts now say when another `quarter` comes first on your PATH.

### Fixed

- `quarter --version` shows the version after `go install` too; it said "dev".

## [0.2.0] - 2026-10-15

### Changed

- Search finds the words you type anywhere in a game's name, in any order, with capitals and punctuation ignored ("spiderman" finds Marvel's Spider-Man), and lists the best matches first: the start of the name, then the start of a word, then inside a word. It used to match letters scattered across a name, so "cyber" also found Black Myth: Wukong Benchmark Tool.
- On Windows, a game played with Steam already open starts without Steam's "Launching..." box: Quarter hands it to Steam as `steam.exe -applaunch`, as it already did with Steam closed, instead of a `steam://` link. If steam.exe can't be run, the link starts the game as before. `"steam_start": "normal"` keeps the link and the box.

## [0.1.0] - 2026-10-10

The first release.

### Library

- Reads your Steam libraries on every drive, with no login: installed games and the ones you own but haven't installed.
- Playtime, last played and achievement progress from Steam's own files.
- Search as you type, sort by name, playtime or last played, filter by All, Installed, Recent and Favourites; favourites are starred and remembered.
- Opens in an instant from the last scan, then rescans in the background; games installed or uninstalled while Quarter is open show up on their own.

### Playing

- Play, install, uninstall (press twice) and the store page, all handed to the Steam client.
- On Windows, Steam starts in the tray when Quarter opens, and a game played with Steam closed starts without Steam's window ("Opening Steam" shows while Steam signs in); `"steam_start": "normal"` keeps Steam's window. macOS and Linux open Steam as before.
- The game running now, and downloads and updates with their progress, shown live.
- A game's logo opens it in your Steam library; a game Steam can't show there gets a message instead.

### Details

- A details pane with developer, year, genres, achievements, size, library folder, publisher and the official website.
- `w` opens the website and `b` the game's SteamDB page; short links are followed to the real site.

### Logos

- Game logos as real pictures in terminals with Sixel graphics (Windows Terminal 1.22 or later and others), as one-colour block art elsewhere.

### History

- Play sessions recorded from Steam's log and kept, so your history outlives it.
- A ranking by period and a HIGH SCORES board with a calendar of played days.
- Past years from Steam Replay, only when you ask for them.

### Arcade

- The INSERT QUARTER start screen with 32 intros, an attract mode while it waits, and 11 goodbyes on quitting.

### Everything else

- Keyboard first, with full mouse support (`"mouse": "off"` leaves it to the terminal).
- Clear messages when Steam isn't found, the library is empty, the store can't be reached or a saved file is damaged.
- Builds for Windows, macOS and Linux, on amd64 and arm64. Tested on Windows 11; the macOS and Linux builds are untested.

[Unreleased]: https://github.com/stronxyo/quarter/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/stronxyo/quarter/releases/tag/v0.2.1
[0.2.0]: https://github.com/stronxyo/quarter/releases/tag/v0.2.0
[0.1.0]: https://github.com/stronxyo/quarter/releases/tag/v0.1.0
