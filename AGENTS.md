# Agentic Kernel

This file **is** the agentic kernel: a platform-agnostic foundation for building stateful, multi-agent systems. It defines the initialization protocol, session contract, and directory structure that any project can adopt and extend. The `.agentic/` directory it points to holds the universal behavioral rules, decision framework, agents, workflows, skills, and memory.

You have no built-in memory between sessions. This system is how you become stateful. Read it to remember, write to it so the next session knows what happened.

## Initialization

On every session start, in order:

1. Read `.agentic/manifest.yml` to understand the project context and what agents, workflows, and skills are available and how they relate to each other
2. Read `.agentic/core/BEHAVIOR.md` to load your universal behavioral rules
3. Read `.agentic/core/DECISIONS.md` to load the decision-making framework
4. Read relevant entries from `.agentic/memories/state/` to orient to current reality
5. Read recent entries from `.agentic/memories/history/` to pick up context from prior sessions

## Session Protocol

### Start
- Complete initialization above before taking any action
- Do not assume state. Verify from memory files.
- An agent's `Activation` field describes when it is relevant. How an agent is invoked is determined by the platform (e.g. GitHub Copilot, Claude). The kernel does not control invocation.

### During
- Follow the behavioral rules in `.agentic/core/BEHAVIOR.md` at all times
- Apply the decision framework from `.agentic/core/DECISIONS.md` when prioritizing or resolving conflicts
- Update `.agentic/memories/state/` when facts about reality change
- Append to `.agentic/memories/history/` with what was done, decisions made, and open items
- Update `manifest.yml` immediately when any agent, workflow, skill, or memory file is added or removed

### End
- Ensure any changed state is reflected in `.agentic/memories/state/`
- Append a session summary to the appropriate `.agentic/memories/history/` file

## Structure

```
.agentic/
├── manifest.yml          # Registry: active agents, workflows, skills, memories
├── core/                 # Universal behavioral rules (the kernel; treat as immutable)
│   ├── BEHAVIOR.md       # Tone, response style, ADHD-friendly rules, memory protocol
│   └── DECISIONS.md      # NEED/WANT/MAY decision framework
├── agents/               # Domain-specific agent identities
│   └── {agent-name}/     #
│       ├── IDENTITY.md   # Required: role, domain, scope, operating principles
│       ├── notes/        # Required: ongoing observations and adjustments
│       ├── memories/     # Required: agent-specific interaction history
│       ├── assets/       # Optional: templates and resources
│       └── references/   # Optional: documentation
├── workflows/            # Sequences of actions for specific tasks
├── skills/               # Capabilities and functionalities
└── memories/
    ├── state/            # Mutable reference facts (org, people, systems)
    └── history/          # Append-only narrative (decisions, reasoning, how we got here)
```
