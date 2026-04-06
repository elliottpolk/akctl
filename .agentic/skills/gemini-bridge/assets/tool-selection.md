# Tool Selection Guide — Gemini Agent Bridge

Use this guide to translate an agent's **Scope** field in `IDENTITY.md` into the correct tool list for the generated agent file.

## Tool Categories

| Category | Tools | Include when… |
|---|---|---|
| **read** | `read_file`, `glob`, `grep_search`, `list_directory` | Agent reads files, searches codebases, or inspects project state |
| **web** | `web_fetch`, `google_web_search` | Agent retrieves external documentation, searches the web, or fetches URLs |
| **write** | `write_file`, `replace` | Agent modifies or creates files |
| **execute** | `run_shell_command` | Agent runs commands, scripts, or terminal operations |
| **orchestrate** | `codebase_investigator`, `generalist`, `cli_help` | Agent spawns sub-agents or performs complex repository-wide analysis |
| **interactive** | `ask_user` | Agent needs to clarify with or get input from the user |

## Scope → Tool Mapping

### Read-only / Analysis / Audit
Scope contains words like: *reads*, *analyzes*, *reports*, *reviews*, *does not modify*, *read-only*

```
tools: ["read_file", "glob", "grep_search", "list_directory", "web_fetch", "google_web_search"]
```

### Content Creation / Documentation
Scope contains words like: *writes*, *generates*, *drafts*, *documents*, but no execution or command-running

```
tools: ["read_file", "glob", "grep_search", "list_directory", "web_fetch", "google_web_search", "write_file", "replace", "ask_user"]
```

### Engineering / Development
Scope contains words like: *implements*, *refactors*, *fixes*, *builds*, *runs tests*, *executes*

```
tools: ["read_file", "glob", "grep_search", "list_directory", "web_fetch", "google_web_search", "write_file", "replace", "run_shell_command", "ask_user"]
```

### Orchestration / Multi-agent
Scope contains words like: *coordinates*, *delegates*, *orchestrates*, *spawns agents*, *manages workflows*

```
tools: ["*"]
```

## Overrides

- If Scope **explicitly states** the agent does not write, modify, or execute — drop `write_file`, `replace`, and `run_shell_command` regardless of category match.
- If Scope **explicitly states** the agent does not run commands or scripts — drop `run_shell_command`.
- If Scope **explicitly states** the agent does not spawn sub-agents — use narrow tool list instead of `*`.
- When in doubt, choose the **narrower** tool set. It is better to under-grant and let the operator expand than to over-grant.

## Note on Experimental Agents
Custom agents in Gemini CLI are an experimental feature. They must be enabled in `settings.json` via:
```json
"experimental": {
  "enableAgents": true
}
```
If not enabled, the generated agent files in `.gemini/agents/` will not be recognized by the CLI.
