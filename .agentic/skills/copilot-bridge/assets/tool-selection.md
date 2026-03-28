# Tool Selection Guide

Use the agent's **Scope** field to determine the appropriate tool set. Select the minimal set that satisfies the agent's responsibilities — do not add tools speculatively.

## Read-only agents
For agents that only read, analyse, navigate, or answer questions (no file writes, no code execution):
```
[vscode, read/readFile, browser, search, web, todo]
```

## Read + write agents
For agents that create or modify files as part of their core responsibilities:
```
[vscode, read/readFile, browser, edit, search, web, todo]
```

## Read + write + execute agents
For agents that also run commands, tasks, or tests:
```
[vscode, execute, read/readFile, agent, browser, edit, search, web, todo]
```

## Decision rule
If the Scope field contains phrases like "does not write", "read-only", "navigational", or "does not execute" — use the read-only set.
If it creates or modifies files — use read + write.
If it runs commands, builds, tests, or deploys — use read + write + execute.
When in doubt, start with a smaller set. Tools can always be added; unnecessary tools expand attack surface.
