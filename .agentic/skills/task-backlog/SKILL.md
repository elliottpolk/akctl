---
name: task-backlog
description: Cross-agent task capture and execution protocol. Use when identifying work to defer during an active session, or when asked to list or work through pending tasks. Any agent can capture; any agent can query and execute.
---

# task-backlog

A protocol for capturing deferred work during any session and working through it later, from any agent.

## Backlog File

Location: `.agentic/memories/state/tasks.md`

## Task Format

```
- [ ] {description}
  - agent: {suggested-agent-name}
  - created: {YYYYMMDD}
  - source: {brief session context — what were you doing when this was identified}
  - context: {optional — detail the target agent needs to execute the task}
```

Status markers:
- `[ ]` open
- `[~]` in progress (claimed by current session)
- `[x]` done
- `[-]` dropped

## Capturing a Task

Trigger: during a session, work is identified that will not be done now.

1. Determine the best `agent:` from the manifest (`.agentic/manifest.yml`). If none fits, set `agent: unassigned`.
2. Write the task entry in the format above.
3. Append it to the `## Open` section of `.agentic/memories/state/tasks.md`.
4. Confirm briefly: "Task captured for {agent-name}: {description}"
5. Continue the current session without interruption.

Do not ask the user to fill out the format. Infer everything from context. Only ask if the description or target agent is genuinely ambiguous.

## Querying Tasks

Trigger: user asks "what tasks do we have", "what's in the backlog", or equivalent.

1. Read `.agentic/memories/state/tasks.md`.
2. List all `[ ]` open tasks, grouped by `agent:` value.
3. Ask: "Which task do you want to work on? I can also route to the suggested agent if we switch modes."

If the backlog is empty, say so and offer to continue current work.

## Working Through a Task

1. Mark the task `[~]` in the backlog file.
2. If the task's `agent:` differs from the current agent, read that agent's `IDENTITY.md` before proceeding so the work is executed in the right frame.
3. Execute the work.
4. On completion: mark `[x]`, add `completed: {YYYYMMDD}`, and move the entry to `## Done`.
5. On drop: mark `[-]`, add `dropped: {YYYYMMDD}` and a one-line reason, and move the entry to `## Dropped`.

## Agent Assignment Reference

Common assignments:
- Work to be done in a daily work log: `work-devlog`
- Changes to the agentic system itself: `kernel`
- New agents to scaffold: `agent-foundry`
- Initiative operationalization: `toph`
- Adversarial review of an initiative: `toph-red-team`
- Personal monthly log work: `personal-devlog`
- Unclear or cross-cutting: `unassigned`
