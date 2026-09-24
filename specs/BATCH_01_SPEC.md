# Batch 1 Spec: Scan

**Branch:** `batch/01-scan`
**Status:** Done (approved 2026-09-24). On Pablo's PC: 2 libraries, 11 games, playtime matches the Steam client within 0.1 h. SteamVR added to the denylist after the real scan.
**Goal:** Quarter finds Steam on its own, reads every library on every drive, and prints your installed games with hours played and last played date. No UI yet: plain text via `quarter --list`.

---

## Scope decisions

- **Installed games only.** Owned-but-not-installed games need the Web API (Batch 6). `localconfig.vdf` has playtime for uninstalled games too, but no names, so they are skipped for now.
- **One new dependency:** `golang.org/x/sys` (locked stack), used only in `locate_windows.go` for the registry.
- **No logging to file yet.** Warnings from `--list` go to stderr. File logging arrives with the TUI (Batch 2), when the terminal is no longer free.
- **The `Game` model stays minimal:** only fields this batch fills. Achievements and store metadata are added in the batches that fetch them.
- **Everything fails soft** (Rules 4.2): a missing or broken file drops that piece of data with a warning; the rest still prints.

---

## Phase A: VDF parser

Valve's text KeyValues format (`.vdf`, `.acf`): quoted keys and values, nested `{ }` blocks, `//` comments.

**Design**
- `vdf.Parse(r io.Reader) (*Node, error)`.
- `Node{Key, Value string; Children []*Node}`. A node has either a value or children.
- `(*Node).Get(key) *Node`: first child with that key, **case-insensitive** (Steam is inconsistent with casing, see Lutris PR #6902). Returns nil if absent, so lookups chain without crashing.
- `(*Node).Path(keys...) *Node`: nested lookup, nil-safe.
- Handles: escapes (`\"`, `\\`, `\n`, `\t`), unquoted tokens, duplicate keys (kept in order), CRLF and LF.
- Malformed input returns an error, never panics.
- Single pass over the input: `localconfig.vdf` can be several MB.

**Files:** `internal/vdf/vdf.go`, `internal/vdf/vdf_test.go`

**Done when:** table-driven tests pass for valid files, every edge case above, and malformed input (unclosed brace, unclosed quote, stray brace).

**Commit**
```
git add internal/vdf/vdf.go internal/vdf/vdf_test.go
git commit -m "vdf: add a parser for Valve's text KeyValues format, since every Steam file Quarter reads uses it."
git push
```

---

## Phase B: Locate Steam, libraries and installed games

**Design**
- **Locate**, first match wins:
  1. `--steam-path` flag.
  2. `QUARTER_STEAM_PATH` environment variable.
  3. Windows: registry `HKCU\Software\Valve\Steam` value `SteamPath`. Other OS: `~/.steam/steam`, `~/.local/share/Steam`, `~/Library/Application Support/Steam` (first that exists).
- **Libraries:** `steamapps/libraryfolders.vdf` gives every library `path`. The Steam install itself is always included, even if missing from the file. Duplicates removed.
- **Installed games:** every `steamapps/appmanifest_*.acf` in each library gives `appid`, `name`, `installdir`, `SizeOnDisk`, `StateFlags`, `LastUpdated`. Installed means the fully-installed bit of `StateFlags` is set.
- **Filter:** drop non-games by appid denylist (Steamworks Common Redistributables, Proton versions, Steam Linux Runtime and similar) kept in `internal/config/constants.go`.

**To verify during this phase** (added to Rules Appendix C with sources before code depends on them):
- `StateFlags` bit meaning for "fully installed" (believed to be value `4`).
- Exact appids for the denylist.
- The non-Windows default Steam paths above.

**Files**
- `internal/steam/locate.go`, `locate_windows.go`, `locate_other.go`
- `internal/steam/libraries.go`, `manifests.go`, `filter.go` (+ `_test.go` for each)
- `internal/config/constants.go` (denylist)
- `testdata/steam/...` (fake install: two libraries, games, one redistributable, one half-installed game, one broken manifest)
- `go.mod`, `go.sum` (`golang.org/x/sys`)

**Done when**
- Tests pass against `testdata/steam/`: both libraries found, redistributable filtered, half-installed and broken manifests handled without crashing.
- On this PC, the scan finds the same number of games as the appmanifest files in your libraries, minus filtered tools.

**Commit**
```
git add go.mod go.sum internal/steam internal/config/constants.go testdata/steam RULES/QUARTER_RULES_V1.md
git commit -m "steam: find the Steam install and read installed games from every library folder, so games on any drive show up."
git push
```

---

## Phase C: Account, playtime and last played

**Design**
- **Account:** `config/loginusers.vdf`. Pick the block with `MostRecent` = 1; if none, the highest `Timestamp` (Rules Appendix C #6). Its key is the SteamID64; `accountID = SteamID64 - 76561197960265728`.
- **Playtime and last played:** `userdata/<accountID>/config/localconfig.vdf`, path `UserLocalConfigStore/Software/Valve/Steam/apps/<appid>`. Keys `Playtime` (minutes) and `LastPlayed` (Unix time). Lookup is case-insensitive.
- If the account or file is missing, games still list, with playtime shown as unknown.

**To verify during this phase:** the SteamID64 to account ID offset, and the exact key names (see open question 1).

**Files:** `internal/steam/users.go`, `localconfig.go` (+ tests), `testdata/steam/config/loginusers.vdf`, `testdata/steam/userdata/...` (fake data only, never copied from this PC)

**Done when**
- Tests pass: `MostRecent` present, `MostRecent` absent (falls back to `Timestamp`), two accounts, missing `localconfig.vdf`.
- On this PC, playtime for a few games matches what the Steam client shows.

**Commit**
```
git add internal/steam/users.go internal/steam/users_test.go internal/steam/localconfig.go internal/steam/localconfig_test.go testdata/steam
git commit -m "steam: read the active account and each game's playtime and last played from local files, so no API key is needed for them."
git push
```

---

## Phase D: `quarter --list`

**Design**
- `internal/library/game.go`: `Game{AppID, Name, Installed, LibraryPath, SizeOnDisk, Playtime, LastPlayed}`.
- `internal/library/merge.go`: joins installed games with playtime data by appid.
- `quarter --list` prints one line per game, sorted by name: name, hours played, last played date (or `-`). Totals line at the end: game count and total hours.
- `quarter --steam-path <dir>` works with `--list`.
- Warnings (missing files) go to stderr, so the list itself stays clean.

**Files:** `internal/library/game.go`, `merge.go`, `merge_test.go`, `cmd/quarter/main.go`

**Done when**
- `quarter --list` prints your real installed games with hours played.
- `quarter --list --steam-path testdata/steam` prints the fixture games.
- `gofmt`, `go vet`, `staticcheck`, `go test ./...` all clean.

**Commit**
```
git add internal/library cmd/quarter/main.go specs/BATCH_01_SPEC.md
git commit -m "cli: add quarter --list to print installed games with playtime, the first real output of the scanner."
git push
```

---

## PR to main

Claude writes the title and description after Phase D. Pablo opens and merges.

## Open questions for Pablo

1. **Key names in `localconfig.vdf`.** To make the test fixture match reality, may I read the structure of **one** game's entry in your `localconfig.vdf`, printing key names only with every value replaced by `<hidden>`? Without it, I build the fixture from public sources and confirm the result only by comparing `--list` with the Steam client.
