# Quarter Rules V1

> **Purpose:** The authoritative rules for Quarter. Every Claude session reads this FIRST, before proposing or coding. Pairs with `PROJECT_BRIEF.md` (what we build) and `PLAN.md` (batches).
>
> **Last updated:** 2026-09-24 (V1 created, adapted from VENDO_RULES_V1 and VERXUS_RULES_V17).
> **Maintained by:** Pablo, with Claude as collaborator.

---

## Table of Contents

0. [Read This First](#0-read-this-first)
1. [Communication](#1-communication)
2. [Engineering Philosophy](#2-engineering-philosophy)
3. [Locked Tech Stack](#3-locked-tech-stack)
4. [Code Conventions](#4-code-conventions)
5. [Steam Boundaries](#5-steam-boundaries)
6. [Batches, Branches and PRs](#6-batches-branches-and-prs)
7. [Pre-Submit Checklist](#7-pre-submit-checklist)
8. [Appendix A: Deferred Features with Triggers](#appendix-a-deferred-features-with-triggers)
9. [Appendix B: Open Decisions](#appendix-b-open-decisions)
10. [Appendix C: Unverified Claims](#appendix-c-unverified-claims)

---

## 0. Read This First

### 0.1 Authority

**Pablo is the authority.** Every markdown file here is reference material, not law. When Pablo says something in-session that contradicts this file, Pablo wins for that session and this file gets updated after.

**Precedence when in doubt:**
1. Pablo, right now, in this session.
2. This file (`RULES/QUARTER_RULES_V1.md`).
3. The current batch spec (`specs/BATCH_XX_SPEC.md`).
4. `PLAN.md`.
5. `PROJECT_BRIEF.md`.
6. Memory entries under `~/.claude/projects/C--PROJECTS-Quarter/memory/`.

If two docs conflict: surface the conflict, propose a resolution, wait for Pablo. Never silently pick one.

### 0.2 Session startup procedure

1. Read `MEMORY.md` (automatic).
2. Read this file.
3. Read the current batch spec. If none exists for the work asked, the first task is writing it.
4. Only then propose or code.

### 0.3 Never modify files without explicit approval

**Approval keywords:** "yes", "dale", "go ahead", "hacelo", "do it", "procede", "make it", or equivalent explicit confirmation of the specific action just proposed.

"Propose", "give me the plan", "how would you do X", "what do you think" means text only. No Edit, Write, `go get`, `go mod tidy` or installs until a keyword arrives.

Read-only operations need no approval: Read, Grep, Glob, WebSearch, WebFetch, `go build`, `go test`, `go vet`, and git commands that only inspect.

**Scope:** one approval covers exactly what was proposed. Related follow-ups need a new approval.

### 0.4 Never run `git commit` or `git push`

Claude NEVER runs `git commit`, `git commit --amend`, `git push`, `git merge` into `main`, or any equivalent. Pablo commits, pushes and merges. Hard boundary, not negotiable.

Claude MAY run read-only git (`status`, `diff`, `log`, `show`, `blame`).

**Close-out for every set of changes** is a ready-to-paste block, naming each file (never `git add .`):

```
git add internal/steam/libraries.go internal/steam/libraries_test.go
git commit -m "steam: read every library folder from libraryfolders.vdf, since games can live on any drive."
git push
```

Commit message style: `scope: what changed, and why.` One sentence, ends with a period. No attribution lines.

### 0.5 When to update this file

- A locked stack choice changes.
- A new convention is agreed.
- A deferred feature's trigger fires (promote it out of Appendix A).
- An open decision in Appendix B is resolved.
- A claim in Appendix C is verified or disproved.

Minor edits in place. Major restructures bump the version (`QUARTER_RULES_V2.md`).

---

## 1. Communication

### 1.1 Language

- **English only.** Replies, code, comments, docs, UI text. Quarter has no localization and no i18n layer.
- If Pablo writes in Spanish, reply in Spanish for that turn. Everything written to the repo stays English.

### 1.2 Terse

- Short answers. No preamble, no recap of what Pablo just said, no filler.
- End-of-turn summary: one or two sentences, plus the git block when there is something to commit.

### 1.3 No em dashes, ever

Never `—` or `–` anywhere: replies, code comments, commit messages, docs, UI text. Use a comma, colon, parentheses, semicolon, or two sentences.

### 1.4 No emojis

Not in replies unless Pablo uses them first. Never in code, comments, commits, docs or UI text.

### 1.5 Complete code, no snippets

- No `// ... existing code`, `// rest stays the same`, `// TODO fill in`.
- Small, well-scoped Edit calls for genuinely one-line changes are fine. The rule is about never handing off half-done code.

### 1.6 Zero assumptions protocol (MANDATORY)

1. **Always cite the source:** `file.go:42`, "you said this in conversation", a URL, or "this is my interpretation". Never blend sources silently.
2. **If not read, say so.** Don't fill gaps with general knowledge stated as fact.
3. **Summarize only what's in the document.**
4. **When unsure, say "I'm unsure."** Silence over invention.
5. **Quote, don't paraphrase**, when the exact wording matters.

---

## 2. Engineering Philosophy

### 2.1 Boring until it breaks

Pick what has survived production. Every novel choice spends from a budget of 3 to 5. Quarter's novel choices: **Bubbletea as a full-screen app on Windows Terminal** and **sixel images** (not yet approved, see Appendix A). Everything else is the boring default.

**Test for every proposal:** "Have I felt this pain, or is it hypothetical?" Hypothetical means defer with a trigger (Appendix A).

### 2.2 D.R.Y. super rule

Before writing any new function:
1. Grep for similar logic.
2. If it exists, reuse it.
3. If it is almost the same, extract the shared part and call it from both places.

**Red flags:** copy-paste; `parseXForLibraries()` next to `parseXForManifests()` with 80% overlap; fixing the same bug twice; an early `return` that blocks a shared code path below it.

> "The best code is NO code. The second best is REUSED code. The worst is DUPLICATE code."

### 2.3 Anti-shortcut verification

- Never guess about code. Every claim about existing code gets `path:line`.
- Every claim about a library version or API gets verified against docs or `go.mod`.
- "I don't know yet, let me check", then check.

### 2.4 Research-backed claims

- Before asserting a library version, API behavior, Steam file format, rate limit or terminal capability: verify via WebSearch or WebFetch, with the current year in the query.
- End any reply that used WebSearch with a `Sources` section.
- Anything stated before verification goes in Appendix C until verified.

### 2.5 Simplicity over cleverness

- The simpler solution wins. One responsibility per function.
- Readable beats clever.
- No abstraction for features that may never come (no plugin system, no interface with one implementation).

### 2.6 Trust existing working code

If the VDF parser reads `libraryfolders.vdf`, it reads `localconfig.vdf` too. Do not write a second parser.

### 2.7 Never break working features

After any change, the previous batches' "Done when" checks still pass.

---

## 3. Locked Tech Stack

**Do not re-litigate without Pablo's explicit approval.** Versions are pinned in Batch 0 after verification.

| Layer | Choice | Notes |
|---|---|---|
| Language | **Go** (latest stable) | Single static binary, no CGO. |
| TUI runtime | **Bubbletea** | Elm-style model/update/view. |
| Components | **Bubbles** | textinput, viewport, key, help. |
| Styling | **Lip Gloss** | All colors and borders as named styles in `internal/ui/styles.go`. |
| Fuzzy search | **sahilm/fuzzy** | |
| Windows registry | **golang.org/x/sys/windows/registry** | Only in `locate_windows.go`. |
| Images | **stdlib `image`, `image/jpeg`, `image/png`** | Half-block renderer is ours. |
| HTTP | **stdlib `net/http`** | Always with a timeout and `context`. |
| JSON cache | **stdlib `encoding/json`** | Files in the app data folder. |
| Logging | **stdlib `log/slog`** | To a file in the app data folder, never to the terminal while the TUI runs. |
| Flags | **stdlib `flag`** | |
| Tests | **stdlib `testing`** + `teatest` for UI flows | Table-driven, against `testdata/`. |
| Lint | **`gofmt`, `go vet`, `staticcheck`** | Must pass before every close-out. |
| Release | **GoReleaser** | Windows/macOS/Linux, amd64 + arm64. |

### 3.1 Anti-list (do not add without Pablo's explicit approval)

cobra, viper, urfave/cli, logrus, zap, zerolog, testify, gomock, resty, any ORM, SQLite (or any database), any CGO dependency, any Steam login or Steam network library (steam-go, go-steam and similar), any GUI toolkit, any i18n library.

### 3.2 Folder structure (locked)

```
cmd/quarter/        # main: flags, wiring
internal/vdf/       # text VDF/ACF parser
internal/steam/     # locate, libraries, manifests, users, localconfig, filter, launch
internal/webapi/    # Steam Web API + store appdetails
internal/library/   # Game model + merge
internal/cache/     # JSON + image cache
internal/config/    # paths, config.json, favourites, constants.go
internal/art/       # art sources + half-block renderer
internal/ui/        # Bubbletea models and views
testdata/steam/     # fake Steam install for tests
specs/              # BATCH_XX_SPEC.md
RULES/              # this file
```

### 3.3 Dependency discipline

- If it is under 20 lines, write it.
- A new dependency needs: active maintenance (commits in the last 6 months), and a check that the stdlib or an existing dependency can't do the job.

---

## 4. Code Conventions

### 4.1 Formatting and lint

`gofmt` (via `goimports`), `go vet ./...` and `staticcheck ./...` clean before every close-out.

### 4.2 Errors

- Return errors, wrapped with context: `fmt.Errorf("read libraryfolders: %w", err)`.
- No `panic` outside `main` startup.
- **Fail soft on Steam's local files.** A bad or missing file disables that feature and logs it; it never crashes the app.

### 4.3 Never block the UI

All disk and network work runs inside a `tea.Cmd` and returns a message. `Update` and `View` never do I/O.

### 4.4 No hardcoded values

- Magic numbers, URLs, timeouts, rate limits and the non-game appid denylist live in `internal/config/constants.go`.
- Colors and borders live in `internal/ui/styles.go`.
- Keybindings live in `internal/ui/keys.go`.

### 4.5 Boundaries get validated

Data from Steam files, the Web API and the store endpoint is untrusted. Parse into typed structs, check required fields, and treat missing fields as "unknown", not as zero.

### 4.6 Naming

Standard Go: `MixedCaps`, short receiver names, package names lowercase single words, no `I` prefix on interfaces, no `Get` prefix on getters. Constants in `MixedCaps` (Go convention, not `UPPER_SNAKE`).

### 4.7 Comments discipline

- Default: no comments.
- Comment only a non-obvious **why**: a Steam format quirk, a workaround, a hidden constraint.
- Exported identifiers get a doc comment only when the package is meant to be reused.
- Never reference the current batch, task or bug in a comment.

### 4.8 Tests

- Every parser in `internal/vdf` and `internal/steam` has table-driven tests against `testdata/`.
- `internal/webapi` is tested with `httptest`, never the real API.
- No test touches the real Steam install, the network, or the user's app data folder.

---

## 5. Steam Boundaries

Non-negotiable, from the brief's core principles.

1. **Never replace Steam.** Actions go through `steam://` URIs only.
2. **Never handle Steam credentials.** No username, password or session token. Ever. The Web API key is the only secret.
3. **Read-only on Steam's folders.** Quarter never writes, renames or deletes anything under the Steam install or `userdata`.
4. **The API key** lives in `config.json` in the app data folder, never in the repo, never in logs.
5. **Respect rate limits.** The store `appdetails` endpoint is fetched for the selected game first and backfilled slowly. It stays optional: no feature may depend on it.
6. **Cover art** is shown from the user's own Steam cache or the CDN. Never bundled, never rehosted.
7. **No "Steam" in the name, logo or branding**, and nothing that implies Valve endorses Quarter.

---

## 6. Batches, Branches and PRs

### 6.1 Batches and phases

- Work happens in numbered **batches** (`PLAN.md`). Each batch is split into **phases**. A phase is one reviewable, runnable step.
- **Spec first.** Before any code in a batch, Claude writes `specs/BATCH_XX_SPEC.md`: goal, phases, files touched, "Done when" per phase, test plan, open questions. Pablo approves the spec before code.
- Nothing outside the approved spec gets built. New ideas go to Appendix A or the next batch.

### 6.2 One branch per batch

- Branch name: `batch/NN-slug` (e.g. `batch/01-scan`), created from an up-to-date `main`.
- Claude gives Pablo the commands; Pablo runs them:
  ```
  git checkout main
  git pull
  git checkout -b batch/01-scan
  ```
- One commit per phase (at least), using the close-out block from 0.4.
- First push of a new branch: `git push -u origin batch/01-scan`.

### 6.3 PR to main

- When every phase's "Done when" passes and section 7 is clean, Claude writes the PR title and description for Pablo (summary, phases, how to test).
- Pablo opens the PR and merges. `main` always builds and runs.

---

## 7. Pre-Submit Checklist

Before every close-out:

### Code quality
- [ ] Complete code (no placeholders).
- [ ] D.R.Y. check done (grepped for similar logic first).
- [ ] Simplest solution chosen; existing patterns followed.
- [ ] `gofmt`, `go vet`, `staticcheck` clean.
- [ ] `go test ./...` passes.

### Behavior
- [ ] No I/O in `Update` or `View`.
- [ ] Steam file failures degrade a feature, never crash.
- [ ] Constants, styles and keys in their files (4.4).
- [ ] Previous batches' "Done when" checks still pass.

### Steam boundaries
- [ ] Nothing written under the Steam folder.
- [ ] No credentials anywhere; API key not in repo or logs.
- [ ] No feature depends on the store endpoint.

### Performance (from the brief)
- [ ] Startup from cache to an interactive list under 100 ms (benchmark, from Batch 2 on).
- [ ] No dropped keystrokes in search on the 5,000-game fixture (from Batch 2 on).

### Writing
- [ ] No em/en dashes, no emojis.
- [ ] Claims cited; unverified claims listed in Appendix C.

### Commit hygiene
- [ ] Git block lists specific files.
- [ ] Message: `scope: what changed, and why.`
- [ ] No secrets, no binaries, no `dist/` staged.

---

## Appendix A: Deferred Features with Triggers

Do not propose building these until the trigger fires.

| Feature | Trigger |
|---|---|
| Sixel cover art | Batch 4 spike shows it renders without flicker or leftovers in Windows Terminal. |
| Read `appinfo.vdf` (binary) | The appid denylist misses non-games in real libraries, or we need app metadata offline. |
| Windows Credential Manager for the API key | A user asks, or before any paid release. |
| Non-Steam shortcuts (`shortcuts.vdf`) | After v0.1 ships. |
| Import Steam favourites | Users ask for it. |
| Store screenshots in the detail pane | After cover art is solid. |
| Profile-guided optimization (PGO) | `pprof` shows a CPU hot spot the code can't fix directly. |
| Epic / GOG | v1.0 per the brief's roadmap. |
| Localization | Never, unless Pablo changes this rule. English only. |

## Appendix B: Open Decisions

- **License:** None for now (all rights reserved, repo private). MIT vs GPLv3 (+ private Pro repo) decided before going public.
- **Setup wizard:** auto on first run, or only via a key. Current default: key only.
- **Repo name casing:** GitHub repo is `stronxyo/Quarter`; module path is `github.com/stronxyo/quarter`. Rename the repo to lowercase or match the module path.

## Appendix C: Unverified Claims

Stated earlier without a checked source. Verify in Batch 0 (with Sources) before any code depends on them.

1. Store `appdetails` rate limit of roughly 200 requests per 5 minutes.
2. Steam Web API limit of 100,000 calls per day, and its commercial-use terms.
3. The Windows Terminal version that added sixel support.
4. `appcache\librarycache\` switched to one subfolder per appid in 2024, and its file names.
5. `localconfig.vdf` holds per-app playtime and last played.
6. `loginusers.vdf` holds SteamID64 with a `MostRecent` flag.
7. `GetOwnedGames` returns empty when the profile's game details are private.
8. Current Steam CDN URL patterns for header and capsule art.
9. Go module paths are case-sensitive in a way that matters for `go install` with the repo's current casing.

---

*End of QUARTER_RULES_V1.md. Pablo commits changes to git (Claude never does; see 0.4).*
