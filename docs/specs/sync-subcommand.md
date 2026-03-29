---
name: "sync-subcommand"
branch: "feature/sync-subcommand"
version: "1.0.0"
status: "draft"
scaffold: true
---

# Spec: Sync Subcommand

## Objective

Updates the core kernel files in an existing `.agentic/` installation by pulling from the repository recorded at `kernel.repository` in `.agentic/manifest.yml`. Preserves all user content and performs a targeted merge: core files are replaced wholesale, upstream-owned entries in `agents:`, `workflows:`, and `skills:` are added if missing without removing user additions, and `project:` and `memories:` are never touched. Also recovers corrupted or partial installs where only one of `AGENTS.md` or `.agentic/` is present.

## Tech Stack

- Go (1.26)
- `github.com/urfave/cli/v2` -- CLI framework; `--force` / `-f` flag
- `github.com/google/go-github/v84` -- GitHub API client
- `golang.org/x/oauth2` -- token-based auth (via `github-api-auth` feature)
- `github.com/charmbracelet/huh` -- interactive confirmation prompts
- `github.com/charmbracelet/lipgloss` -- terminal styling
- `gopkg.in/yaml.v3` -- manifest parsing and merge

## Project Structure

- `cmd/main.go` -- wire the `sync` command action here
- `internal/kernel/` -- reuse existing fetch and parse logic; extend to support manifest merge
- `internal/sync/` -- sync orchestration: state detection, cache management, core file update, manifest merge, rollback

## Scope Boundaries

- MUST always:
  - Detect the installation state before any network call using the table below; abort to `init` only when recovery is not possible
  - Resolve the upstream repository from `kernel.repository` in `manifest.yml`, or fall back to the `repository` field in `AGENTS.md` frontmatter if `manifest.yml` is missing
  - Verify upstream is reachable before any write; abort and leave all files untouched if the fetch fails for any reason
  - Cache all files that will be modified to the OS temp directory before any write begins
  - Warn the user that core file updates are destructive and local modifications to those files will be lost; issue this warning in all cases regardless of whether local changes are detected
  - Display a summary of what will be changed before writing
  - Identify core files from the upstream tree and update only those; never infer coreness from local state alone
  - Replace the `kernel:` block in `manifest.yml` wholesale from upstream
  - Add upstream-defined entries to `agents:`, `workflows:`, and `skills:` in `manifest.yml` if not already present; never remove existing entries
  - Resolve the GitHub API token using the standard chain: `--github.token` flag, then `GITHUB_TOKEN` env var, then interactive prompt, then unauthenticated fallback
  - Check the GitHub API rate limit before any API call; abort with a clear error including the reset time if the limit is exhausted
- MUST ask before:
  - Proceeding with any write when core files will be overwritten (skipped if `--force` / `-f`)
  - Proceeding with recovery mode when a partial install is detected; confirm the user wants sync to repair, not just update (skipped if `--force` / `-f`)
- MUST never:
  - Touch `project:` or `memories:` in `manifest.yml`
  - Remove user-added entries from `agents:`, `workflows:`, or `skills:` in `manifest.yml`
  - Write any file before a successful upstream fetch
  - Abort to `init` when the install is partial but the available artifact is structurally valid (recovery applies instead)
  - Log, print, or persist the GitHub API token value
  - Attempt to contact any non-`github.com` host

## Installation State Detection

| State | Action |
|---|---|
| No `AGENTS.md`, no `.agentic/` | Abort; direct to `akctl init` |
| `AGENTS.md` invalid structure, no `.agentic/` | Abort; direct to `akctl init` |
| `.agentic/` invalid or missing structure, no `AGENTS.md` | Abort; direct to `akctl init` |
| Valid `AGENTS.md` only (no `.agentic/`) | Read `repository` from `AGENTS.md` frontmatter; recover `.agentic/` from upstream |
| Valid `.agentic/` only (no `AGENTS.md`) | Read `kernel.repository` from `manifest.yml`; recover `AGENTS.md` from upstream |
| Both present (normal case) | Read `kernel.repository` from `manifest.yml`; sync core files |
| Version mismatch between `AGENTS.md` and `manifest.yml` `kernel.version` | Treat as normal sync; both are updated to the upstream version |

## Core Files (sync overwrites)

- `AGENTS.md`
- `.agentic/core/BEHAVIOR.md`, `DECISIONS.md`, `MEMORY.md`
- `.agentic/agents/kernel/IDENTITY.md`
- `.agentic/agents/agent-foundry/IDENTITY.md` and `assets/`
- `.agentic/skills/claude-bridge/`, `copilot-bridge/`, `create-skill/`, `task-backlog/` (full directories)
- `.agentic/workflows/spec/`, `scaffold/` (full directories)
- `manifest.yml` `kernel:` block only

Anything else in `.agentic/` (agent `memories/`, `notes/`, `references/`, user-added agents, workflows, skills) is user-owned and MUST NOT be modified.

## Acceptance Criteria

### AC 1: Normal Sync

- **Given** `AGENTS.md` and `.agentic/manifest.yml` both exist and are structurally valid
- **When** the user runs `akctl sync` and confirms the core overwrite warning
- **Then** upstream core files are fetched from `kernel.repository`
- **And** core files are overwritten with upstream versions
- **And** the `kernel:` block in `manifest.yml` is replaced with upstream values
- **And** upstream-defined `agents:`, `workflows:`, and `skills:` entries are added to `manifest.yml` if missing
- **And** user-added entries in those sections are preserved
- **And** `project:` and `memories:` are untouched

### AC 2: Fetch Failure Aborts Sync

- **Given** the upstream repository is unreachable for any reason
- **When** the user runs `akctl sync`
- **Then** the command aborts with a clear error identifying the cause
- **And** no files are modified

### AC 3: Core Overwrite Warning

- **Given** a valid installation
- **When** the user runs `akctl sync` without `--force`
- **Then** a warning is displayed that core file updates are destructive and local modifications will be lost
- **And** a summary of files that will be changed is displayed
- **And** explicit confirmation is required before any write proceeds

### AC 4: Force Flag Skips Confirmation

- **Given** a valid installation
- **When** the user runs `akctl sync --force` (or `-f`)
- **Then** the warning and file summary are still displayed
- **And** confirmation is skipped and sync proceeds immediately

### AC 5: Recovery from Partial Install (valid `AGENTS.md` only)

- **Given** `AGENTS.md` exists and is structurally valid, but `.agentic/` is missing
- **When** the user runs `akctl sync`
- **Then** `repository` is read from `AGENTS.md` frontmatter
- **And** the user is informed of the partial install and asked to confirm recovery
- **And** `.agentic/` is restored from upstream on confirmation

### AC 6: Recovery from Partial Install (valid `.agentic/` only)

- **Given** `.agentic/` and `manifest.yml` exist and are structurally valid, but `AGENTS.md` is missing
- **When** the user runs `akctl sync`
- **Then** `kernel.repository` is read from `manifest.yml`
- **And** `AGENTS.md` is restored from upstream

### AC 7: Abort to Init (no valid artifacts)

- **Given** neither `AGENTS.md` nor `.agentic/` exist, or both are structurally invalid
- **When** the user runs `akctl sync`
- **Then** the command aborts with a message directing the user to run `akctl init`
- **And** no network calls are made

### AC 8: Version Mismatch Resolved by Sync

- **Given** the local `AGENTS.md` version and `manifest.yml` `kernel.version` differ
- **When** the user runs `akctl sync`
- **Then** sync proceeds as normal without aborting
- **And** both are updated to the upstream version

### AC 9: Mid-Sync Failure Rolls Back

- **Given** sync has begun writing files
- **When** any write operation fails (network error, disk full, I/O error, or any other cause)
- **Then** all modified files are restored from the pre-sync cache in the OS temp directory
- **And** the cache is deleted after rollback completes
- **And** the command exits with a clear error identifying the cause

### AC 10: Cache Deleted on Success

- **Given** sync completes successfully
- **When** all writes are confirmed
- **Then** the pre-sync cache in the OS temp directory is deleted

### AC 11: Rate Limit Aborts Sync

- **Given** the GitHub API rate limit is exhausted before any API call
- **When** the user runs `akctl sync`
- **Then** the command aborts with a clear error including the rate limit reset time
- **And** no files are modified

## Open Questions

- GitHub API auth (`--github.token` flag, `GITHUB_TOKEN` env var, interactive prompt) depends on the `github-api-auth` feature being complete. `sync` inherits that implementation; no new auth work is required here.
