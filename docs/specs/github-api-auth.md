---
name: "github-api-auth"
branch: "feature/github-api-auth"
version: "1.0.0"
status: "draft"
scaffold: true
---

# Spec: GitHub API Auth

## Objective

Provides authenticated access to the GitHub API so `akctl` can fetch kernels from private repositories and avoid unauthenticated rate limits that would otherwise block or degrade the init flow.

## Tech Stack

- Go (1.26)
- `github.com/urfave/cli/v2` -- `--kernel.source` / `--kernel.src` flag (no scheme required), `--github.token` flag with `GITHUB_TOKEN` env var fallback
- `github.com/google/go-github/v84` -- GitHub API client (auth transport layer)
- `golang.org/x/oauth2` -- token-based auth for the GitHub client (not yet in go.mod)
- `github.com/charmbracelet/huh` -- interactive token prompt if neither flag nor env var is set
- `github.com/stretchr/testify/assert` -- test assertions

## Project Structure

- `internal/github/` -- GitHub client construction, token resolution, rate limit checking, and host validation

## Scope Boundaries

- MUST always:
  - Check for a token in this order: `--github.token` flag, then `GITHUB_TOKEN` env var, then interactive prompt via `huh`
  - Treat an empty or whitespace-only token as absent and fall through to the next resolution step
  - Use the token to construct an authenticated `go-github` client via `golang.org/x/oauth2`
  - Fall back to an unauthenticated client if no token is provided
  - Parse the host/domain from `--kernel.source`; if it is not `github.com`, abort with a clear error stating the provider is not yet supported
  - Default `--kernel.source` to `github.com/elliottpolk/agentic-kernel` if not provided
  - Check the GitHub API rate limit before making any API calls; abort with a clear error including the reset time if the limit is exhausted, regardless of auth state
- MUST ask before:
  - Nothing specific -- token input via prompt is already interactive by nature
- MUST never:
  - Log, print, or persist the token value anywhere (env var name is fine, the value is not)
  - Store the token to disk in any file `akctl` writes
  - Attempt to contact any non-`github.com` host, even if valid URL syntax is provided

## Acceptance Criteria

### AC 1: Token Resolved from Flag

- **Given** `--github.token` is provided on the command line
- **When** the GitHub client is constructed
- **Then** the provided token is used for authentication

### AC 2: Token Resolved from Env Var

- **Given** `--github.token` is not provided but `GITHUB_TOKEN` is set in the environment
- **When** the GitHub client is constructed
- **Then** the env var value is used for authentication

### AC 3: Token Resolved via Interactive Prompt

- **Given** neither `--github.token` nor `GITHUB_TOKEN` is set
- **When** the GitHub client is constructed
- **Then** the user is prompted interactively for a token via `huh`
- **And** an empty or whitespace-only entry is treated as absent

### AC 4: Unauthenticated Fallback

- **Given** no token is provided at any resolution step
- **When** the GitHub client is constructed
- **Then** an unauthenticated client is used and the operation proceeds

### AC 5: Rate Limit Check Before API Calls

- **Given** any GitHub API call is about to be made (authenticated or not)
- **When** the rate limit is exhausted
- **Then** the command aborts with a clear error including the reset time
- **And** no API calls are made after detecting exhaustion

### AC 6: Inaccessible Repo Prompts Auth Suggestion

- **Given** no token is provided
- **When** the API call returns `404`
- **Then** the command aborts with an error noting the repo may not exist or may be private
- **And** the error suggests providing `--github.token` if the repo is private

### AC 7: Repo Not Found with Token

- **Given** a token is provided
- **When** the API call returns `404`
- **Then** the command aborts with a clear error stating the repo was not found

### AC 8: Default Kernel Source

- **Given** `--kernel.source` is not provided
- **When** the GitHub client is constructed
- **Then** `github.com/elliottpolk/agentic-kernel` is used as the source

### AC 9: Unsupported Host Rejected

- **Given** `--kernel.source` is provided with a non-`github.com` host (e.g., `gitlab.com/...`)
- **When** the command parses the flag value
- **Then** the command aborts with a clear error stating the provider is not yet supported
- **And** no API calls are made

## Open Questions

- Support for other SCM providers (GitLab, Bitbucket, etc.) is explicitly deferred as a future feature.
- Whether `--kernel.source` and `--github.token` should be global flags available to all subcommands (rather than scoped to `init`) is unresolved; global is preferred if `urfave/cli/v2` supports it cleanly without side effects.
- `--kernel.source` env var counterpart (e.g., `AKCTL_KERNEL_SOURCE`) is deferred; the flag is sufficient for now.
