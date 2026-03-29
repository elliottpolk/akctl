# Project: akctl

## Purpose
CLI tool for initializing and maintaining the agentic kernel (`.agentic/`) in any project or directory. Pulls the canonical kernel structure from [elliottpolk/agentic-kernel](https://github.com/elliottpolk/agentic-kernel) and keeps the core elements up to date without overwriting user additions.

## Repository
https://github.com/elliottpolk/akctl

## Tech Stack
- Language: Go (1.26)
- Module: `github.com/elliottpolk/akctl`
- CLI framework: `github.com/urfave/cli/v2`

## Structure
- `cmd/main.go` — entrypoint; defines the CLI app and commands
- `internal/` — internal packages (empty, to be filled in)

## Commands
| Command | Status | Description |
|---|---|---|
| `init` / `setup` | stubbed | Initialize a project/directory with the agentic kernel |

## Status
Early development. `init` command is stubbed and not yet implemented.

## Copyright
© 2026 The Karoshi Workshop
