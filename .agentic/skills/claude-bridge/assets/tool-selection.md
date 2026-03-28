# Tool Selection Guide — Claude Agent Bridge

Use this guide to translate an agent's **Scope** field in `IDENTITY.md` into the correct tool list for the generated skill file.

## Tool Categories

| Category | Tools | Include when… |
|---|---|---|
| **read** | `Read`, `Glob`, `Grep` | Agent reads files, searches codebases, or inspects project state |
| **web** | `WebFetch`, `WebSearch` | Agent retrieves external documentation, searches the web, or fetches URLs |
| **write** | `Edit`, `Write` | Agent modifies or creates files |
| **execute** | `Bash` | Agent runs commands, scripts, or terminal operations |
| **delegate** | `Agent` | Agent spawns sub-agents or delegates tasks to other agents |

## Scope → Tool Mapping

### Read-only / Analysis / Audit
Scope contains words like: *reads*, *analyzes*, *reports*, *reviews*, *does not modify*, *read-only*

```
tools: [Read, Glob, Grep, WebFetch, WebSearch]
```

### Content Creation / Documentation
Scope contains words like: *writes*, *generates*, *drafts*, *documents*, but no execution or command-running

```
tools: [Read, Glob, Grep, WebFetch, WebSearch, Edit, Write]
```

### Engineering / Development
Scope contains words like: *implements*, *refactors*, *fixes*, *builds*, *runs tests*, *executes*

```
tools: [Read, Glob, Grep, WebFetch, WebSearch, Edit, Write, Bash]
```

### Orchestration / Multi-agent
Scope contains words like: *coordinates*, *delegates*, *orchestrates*, *spawns agents*, *manages workflows*

```
tools: [Read, Glob, Grep, WebFetch, WebSearch, Edit, Write, Bash, Agent]
```

## Overrides

- If Scope **explicitly states** the agent does not write, modify, or execute — drop `Edit`, `Write`, and `Bash` regardless of category match.
- If Scope **explicitly states** the agent does not run commands or scripts — drop `Bash`.
- If Scope **explicitly states** the agent does not spawn sub-agents — drop `Agent`.
- When in doubt, choose the **narrower** tool set. It is better to under-grant and let the operator expand than to over-grant.

## Note on MCP / Connector Tools

Do not include MCP-provided tools (e.g. Slack, GitHub, Linear) in the generated skill file. Those are wired at the project level via `.claude/` or `.mcp.json` configuration, not at the skill level.
