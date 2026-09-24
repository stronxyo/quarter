# Batch 0 Spec: Setup and verification

**Branch:** `batch/00-setup`
**Status:** Done (approved 2026-09-24). Go 1.27.0 and staticcheck 2026.2.1 installed; Phase C results are in Rules Appendix C.
**Goal:** a Go toolchain on this machine, a minimal runnable `quarter` binary, and every unverified claim in Rules Appendix C checked against a real source before Batch 1 depends on it.

---

## Scope decisions

- **No third-party dependencies in this batch.** Bubbletea, Bubbles, Lip Gloss and sahilm/fuzzy are added in the batch that first imports them (Batch 2). Adding them now would be undone by `go mod tidy`, since no code uses them yet. `PLAN.md` Batch 0 Phase A is updated to match.
- **No empty package folders.** Only folders that hold code are created. The rest of the structure in Rules 3.2 appears batch by batch.
- **Version comes from the build**, not a hardcoded string: `var version = "dev"` in `main`, overridden by `-ldflags "-X main.version=..."` at release (Batch 8).

---

## Phase A: Toolchain

**Steps**
1. Install Go: `winget install GoLang.Go`.
2. Confirm the installed version is the current stable release (checked against go.dev, cited in Phase C).
3. Install staticcheck at a pinned version: `go install honnef.co/go/tools/cmd/staticcheck@<version>`. The version is chosen in Phase C and recorded in the rules.

**Files touched:** none in the repo.

**Done when**
- `go version` prints the current stable Go.
- `staticcheck -version` runs.

**Commit:** none (nothing in the repo changes).

---

## Phase B: Skeleton

**Steps**
1. `go mod init github.com/stronxyo/quarter`.
2. `cmd/quarter/main.go`: parses flags with stdlib `flag`; `--version` prints `quarter <version>` and exits 0; with no flags, prints a one-line placeholder and exits 0.
3. `internal/config/constants.go`: `AppName = "quarter"` (used for the app data folder name from Batch 1 on).

**Files touched**
- `go.mod` (new)
- `cmd/quarter/main.go` (new)
- `internal/config/constants.go` (new)

**Done when**
- `go run ./cmd/quarter --version` prints `quarter dev`.
- `gofmt -l .` prints nothing; `go vet ./...` and `staticcheck ./...` are clean.
- `go build ./cmd/quarter` produces `quarter.exe`, and `git status` does not show it (covered by `.gitignore`).

**Commit (Pablo runs)**
```
git add go.mod cmd/quarter/main.go internal/config/constants.go
git commit -m "setup: add the Go module and a minimal quarter binary with --version, so every later batch builds on a runnable base."
git push
```

---

## Phase C: Verify claims

Research every item below with WebSearch/WebFetch (current year in queries). For each: **verified**, **wrong** (with the correct fact), or **unverifiable** (with what we do instead). Results replace Rules Appendix C, each with its source link. Corrections flow into `PROJECT_BRIEF.md`.

| # | Claim | Why it matters |
|---|---|---|
| 1 | Store `appdetails` rate limit is about 200 requests per 5 minutes. | Batch 5 rate limiter. |
| 2 | Steam Web API limit is 100,000 calls/day; terms allow this use. | Batch 6, monetization. |
| 3 | Which Windows Terminal version supports sixel. | Batch 4 spike. |
| 4 | `appcache\librarycache\` uses one subfolder per appid since 2024; file names. | Batch 7 cover art. |
| 5 | `localconfig.vdf` holds per-app playtime and last played. | Batch 1 Phase C. |
| 6 | `loginusers.vdf` holds SteamID64 with a `MostRecent` flag. | Batch 1 Phase C. |
| 7 | `GetOwnedGames` returns empty for private game details. | Batch 6 Phase B. |
| 8 | Current Steam CDN URL patterns for header and capsule art. | Batch 7 Phase A. |
| 9 | Module path casing vs repo name `stronxyo/Quarter`: does `go install github.com/stronxyo/quarter@latest` work? | Release, and the rename decision in Appendix B. |
| 10 | **New:** Current major versions and import paths of Bubbletea, Bubbles and Lip Gloss (a v2 line may be current). | Locks the stack before Batch 2. |
| 11 | **New:** Current stable Go version and staticcheck version compatible with it. | Phase A pins. |

**Local checks (read-only, only if Pablo allows it):** claims 4, 5 and 6 can be confirmed by reading the file layout on this PC. The permission classifier blocked this earlier because `userdata` sits next to account data. If Pablo does not allow it, those claims are verified from public sources only and re-checked against `testdata/` fixtures in Batch 1.

**Files touched**
- `RULES/QUARTER_RULES_V1.md` (Appendix C results; stack versions in section 3; Appendix B if claim 9 forces a decision)
- `PROJECT_BRIEF.md` (corrections only)
- `PLAN.md` (Batch 0 Phase A wording)

**Done when**
- Every row above is marked verified, wrong or unverifiable, with a source.
- No claim marked wrong still appears as fact in the brief or rules.

**Commit (Pablo runs)**
```
git add RULES/QUARTER_RULES_V1.md PROJECT_BRIEF.md PLAN.md specs/BATCH_00_SPEC.md
git commit -m "docs: verify the brief's Steam, terminal and toolchain claims against sources, so Batch 1 builds on checked facts."
git push
```

---

## PR to main

When all three phases are done, Claude writes the PR title and description (summary, phases, how to test). Pablo opens and merges it.

## Open questions for Pablo

1. Allow a **read-only** look at the Steam folder layout on this PC (file names only, no file contents from `userdata`) to confirm claims 4 to 6?
