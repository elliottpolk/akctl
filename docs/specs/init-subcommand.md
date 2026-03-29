---
name: "init-subcommand"
branch: "feature/init-subcommand"
version: "1.0.0"
status: "final"
scaffold: true
---

# Spec: Init Subcommand

## Objective

Initializes a project or directory with the agentic kernel by pulling the canonical `.agentic/` structure and `AGENTS.md` from `elliottpolk/agentic-kernel`. `AGENTS.md` serves as the boot entry point that instructs agents how to load and use the kernel.

## Tech Stack

- Go (1.26)
- `github.com/urfave/cli/v2` -- CLI framework
- `github.com/google/go-github/v84` -- GitHub API client
- `charm.land/bubbletea/v2` -- interactive TUI
- `github.com/charmbracelet/huh` -- forms and input prompts
- `github.com/charmbracelet/lipgloss` -- terminal styling

## Project Structure

- `cmd/main.go` -- existing entrypoint; wire the implemented `init` action here
- `internal/kernel/` -- fetch and parse the upstream kernel from GitHub; extract version from `AGENTS.md` frontmatter
- `internal/setup/` -- init orchestration logic; metadata collection, conflict detection, write sequencing

## Scope Boundaries

- MUST always:
  - Check for existing `.agentic/` and `AGENTS.md` before any write
  - Verify connectivity to the upstream kernel and confirm the target repo is reachable before any destructive action; abort with a clear error and leave all existing files untouched if the fetch fails for any reason (network error, repo not found, rate limited, etc.)
  - Pull the kernel version from the upstream `AGENTS.md` frontmatter
  - Write the `kernel:` block in `manifest.yml` from upstream data; never accept user input for it
  - Prompt for project metadata (name, description, author, organization, copyright, license, repository) to populate `manifest.yml` `project:` block
  - Default `project.name` to the current directory name in kebab-case if the user leaves it blank
  - Display the list of files and directories that will be destroyed before any overwrite, even when `--force` is set
- MUST ask before:
  - Overwriting `AGENTS.md` -- warn explicitly that this is destructive and irreversible; require deliberate confirmation (skipped if `--force` / `-f` is set)
  - Overwriting `.agentic/` -- warn explicitly that this is destructive and irreversible; require deliberate confirmation (skipped if `--force` / `-f` is set)
- MUST never:
  - Delete or overwrite `AGENTS.md` or `.agentic/` before the upstream kernel content has been successfully fetched
  - Proceed with any write operations if the upstream fetch fails for any reason
  - Proceed with a destructive overwrite without explicit user confirmation, unless `--force` / `-f` is present
- MAY:
  - Accept `--force` / `-f` flag to skip destructive confirmation prompts, implying yes to all overwrites

## Acceptance Criteria

### AC 1: Fresh Init

- **Given** no `.agentic/` or `AGENTS.md` exists in the target directory
- **When** the user runs `akctl init`
- **Then** the upstream kernel is fetched, project metadata is collected interactively, and `.agentic/` and `AGENTS.md` are written to the target directory
- **And** `manifest.yml` contains the upstream `kernel:` block and the user-provided `project:` block

### AC 2: Fetch Failure Aborts Init

- **Given** the upstream kernel is unreachable (no network, repo not found, rate limited, or any other error)
- **When** the user runs `akctl init`
- **Then** the command aborts with a clear error message identifying the cause
- **And** no files are created, modified, or deleted

### AC 3: Existing Artifacts Prompt for Confirmation

- **Given** `.agentic/` or `AGENTS.md` already exists
- **When** the user runs `akctl init` without `--force`
- **Then** the command displays the list of files and directories that will be destroyed
- **And** warns that the overwrite is destructive and irreversible
- **And** requires explicit confirmation before proceeding

### AC 4: Force Flag Skips Confirmation

- **Given** `.agentic/` or `AGENTS.md` already exists
- **When** the user runs `akctl init --force` (or `-f`)
- **Then** confirmation prompts are skipped
- **And** the list of files and directories that will be destroyed is still displayed
- **And** existing artifacts are overwritten after a successful upstream fetch

### AC 5: Project Name Defaults to Directory Name

- **Given** the user leaves the project name blank during metadata collection
- **When** `manifest.yml` is written
- **Then** `project.name` is set to the current directory name converted to kebab-case

### AC 6: Project Metadata Written to Manifest

- **Given** a fresh init or confirmed overwrite
- **When** the user completes the metadata prompts
- **Then** `manifest.yml` `project:` block contains the provided values for name, description, author, organization, copyright, license, and repository

### AC 7: Insufficient Permissions Aborts Init

- **Given** the user does not have write permission to the target directory
- **When** `akctl init` attempts to write any file
- **Then** the command aborts with a clear error identifying the permission issue
- **And** any files written before the failure are cleaned up

### AC 8: Mid-Write Failure Cleans Up

- **Given** a write operation begins successfully but fails partway through (e.g., disk full, I/O error)
- **When** the error is detected
- **Then** the command aborts with a clear error
- **And** any files and directories created during that run are removed
- **Note** files deleted as part of a confirmed overwrite before the failure are not restored; the user was warned prior to confirmation

## Open Questions

- Non-interactive flags for pre-populating project metadata (e.g., `--name`, `--author`) are deferred to a future feature for scripted or shorthand use.
- GitHub API token and auth support are deferred until the "bring your own kernel" feature is added (needed for private repos).
