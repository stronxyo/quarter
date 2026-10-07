# Steam starts quietly

Pablo asked for this on 2026-10-07 ("we will need that silent start"), to be built **before the v0.1 release**, in its own session (the release work, Phase D of `specs/BATCH_08_SPEC.md`, waits or runs beside it). Read the project memory first, as `CLAUDE.md` says.

## The problem

Most Steam games need the Steam client running: they check through Steamworks and close, or relaunch through Steam, without it. So Quarter can't play them with Steam closed, and no login inside Quarter would change that (talked through 2026-10-07; DRM workarounds are out, always).

Today, when Steam is closed and the user plays a game from Quarter, the `steam://rungameid/<id>` link starts Steam with its full window, which then sits on top of the terminal. The goal: Steam starts in the tray, without its window, and the game starts.

## How Quarter launches today

- `internal/steam/launch.go`: `PlayURI` builds `steam://rungameid/<id>`.
- `internal/steam/launch_windows.go`: `OpenURI` hands the link to Windows with `ShellExecute` (no console flash).
- `internal/steam/launch_other.go`: `OpenURI` runs `open` (macOS) or `xdg-open` (Linux).
- `internal/steam/running_windows.go`: the running game comes from `HKCU\Software\Valve\Steam\RunningAppID`. `running_other.go`: unknown on Linux and macOS (Steam for Linux stopped writing it in 2023).
- Other links go through the same `OpenURI`: install (`steam://install`), uninstall, the logo click (`steam://nav/games/details/<id>`). Those *should* show Steam's window (the user asked to see Steam), so the quiet start is for play only, unless research says otherwise.

## Research first (nothing assumed, every claim with a source)

1. Valve's documented command-line options (the Valve Developer Community page "Command line options", Steam client section): does `-silent` start Steam in the tray with no window? Does it hold on Windows, macOS and Linux? Any newer option?
2. How to tell, on each system, that Steam is already running (then nothing changes: the link goes straight to it). Windows candidates to verify: Steam's process, or a value under `HKCU\Software\Valve\Steam` (for example `ActiveProcess\pid`; unverified). Linux and macOS: the process, or Steam's own pid file (unverified).
3. How to start Steam with `-silent` on each system: the exe path on Windows (from the registry Quarter already reads in `locate_windows.go`), `open -a Steam --args -silent` on macOS, `steam -silent` on Linux, and the Flatpak (`flatpak run com.valvesoftware.Steam -silent`) and Snap installs. Which of these are documented.
4. Whether a `steam://rungameid` link sent while Steam is still starting (silently) is queued or lost; how long to wait, and how to know Steam is ready.
5. Whether `-silent` leaves Steam's window hidden for good after that start (the user can still open it from the tray) and whether it changes any Steam setting (it must not).

## Rules

- The most universal way that works (Pablo's mandatory rule): documented and cross-platform first; where a system has no quiet start, keep today's behaviour as the fallback, and say so.
- Never change Steam's own settings or files. No DRM workarounds.
- Experiment in a lab copy first (`Documents/Quarter/<name> lab`), never on Pablo's real Steam without his say; he does the live checks.
- A test that fails without the change; speed targets unchanged (`TestSpeedTargets`).

## Open, for Pablo

- Start Steam quietly only when a game is played (the default idea), or as soon as Quarter opens (games then start faster, but Steam runs even if he only browses)?
- A setting to turn it off (`"steam start": "quiet" | "normal"` or similar)?
- When it's done: spec section in `specs/BATCH_08_SPEC.md`, README line if user-facing, CHANGELOG entry under 0.1.0.
