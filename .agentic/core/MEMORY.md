# Memory Field Guide

How agents write, update, and organize memory files. Apply this before writing anything to `.agentic/memories/`.

## The Two Buckets

| Bucket | Path | Rule |
|---|---|---|
| State | `.agentic/memories/state/` | Mutable reference facts. Update in place when reality changes. |
| History | `.agentic/memories/history/` | Append-only narrative. Never edit past entries. |

## State: When and What to Write

Write to `state/` when you learn a fact that would be useful to **any** future agent session:

- Who someone is, what their role is, and how to engage them effectively
- Org structure, team boundaries, governance bodies
- Decisions that are now settled (not pending)
- System facts: what a tool does, how a process works, what the constraints are

Do **not** write to `state/` when:

- The fact is a pattern specific to how one agent reasons or has been corrected (write to that agent's `memories/` directory instead)
- The content is intended for a human to read and act on (write to a human-facing doc like a README or brief)
- The fact is ephemeral or only meaningful within a single session

## History: When and What to Write

Append to `history/` when:

- A decision was made and the reasoning should be preserved
- Something was tried and failed in a non-obvious way
- The current state came to be through a path a future agent would need to understand

Never modify past history entries. Append only. Use a date header for each entry:

```markdown
## 2026-03-27

What was decided and why. What was tried. What the open items are.
```

## File Management

### Naming Conventions

| Subject type | Convention | Example |
|---|---|---|
| Person | `{first-last}.md` | `ahmed-silwadi.md` |
| Org, team, or system | Short descriptive slug | `ets-operating-model.md` |
| Governance body | Short name slug | `garb.md` |
| History (monthly) | `{YYYY-MM}.md` | `2026-03.md` |
| History (topic thread) | `{topic-slug}.md` | `scm-initiative.md` |

### One file per entity, not per session

Do not create a new file for each interaction with a person or system. Add to the existing file. Create a new file only when the subject is genuinely distinct from everything that exists.

### Updating state files

State files reflect **current reality only**. When reality changes:

- Replace the outdated content in place
- Update the `Last updated:` field at the top
- Do not append a changelog of old values -- history files serve that purpose

## General vs. Agent-Specific Memory

Before writing, ask: "Would any agent, with no prior context, need this?"

- **Yes** -- write to `.agentic/memories/state/` or `.agentic/memories/history/`
- **No** -- write to the relevant agent's own `memories/` directory

Agent-specific memory is for how an agent has been shaped over time: corrections, confirmed preferences, agent-scoped patterns. It is not for domain facts.

## Anti-Patterns

- **Writing futures as facts**: Do not write "will" or "plans to" as settled state. Use "as of {date}: pending" or record it in history.
- **Behavioral rules in state**: If it describes how an agent should act, it belongs in IDENTITY.md or BEHAVIOR.md, not state memory.
- **Human-addressed content**: If you would write "you should know..." or structure it as a briefing, it is not agent memory. Write it elsewhere.
- **Over-granularity**: One file per person or system is correct. One file per meeting or interaction is not.
- **Stale orphans**: If a person leaves, a project ends, or a system is decommissioned, mark the file inactive or delete it. Do not let dead state persist.
- **Mixing state and history in one file**: If you need to record both current facts and how they came to be, split them: update the state file and append to a history file.
