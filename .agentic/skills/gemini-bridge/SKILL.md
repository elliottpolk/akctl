---
name: gemini-bridge
description: >-
  Bridges .agentic agents, skills, and workflows into Gemini CLI. Produces
  thin platform-specific wrapper files that reference canonical sources rather than
  duplicating them. Use when making any .agentic component available in Gemini CLI.
compatibility: Gemini CLI
---

# gemini-bridge

Produces Gemini wrapper files from components defined in `.agentic/`. No content is
copied from source files — all generated files reference back to their canonical source.

## Workspace Initialization

Before bridging specific components, ensure the workspace is natively configured for Gemini CLI:

1. Check if `GEMINI.md` exists in the repository root.
2. If it does not exist, create a symlink: `ln -s AGENTS.md GEMINI.md`.
3. If it exists but is not a symlink to `AGENTS.md`, warn the user to resolve the conflict manually.

## Artifact Types and Output Paths

| Input | Source | Output |
|---|---|---|
| Agent | `.agentic/agents/{name}/IDENTITY.md` | `.gemini/agents/{name}.md` |
| Skill | `.agentic/skills/{name}/SKILL.md` | `.gemini/skills/{name}/SKILL.md` |
| Workflow | `.agentic/workflows/{name}/WORKFLOW.md` | `.gemini/commands/{name}.toml` |

## Inputs

- **Type**: one of `agent`, `skill`, or `workflow`
- **Name**: the directory name under the relevant `.agentic/` subdirectory

## Process: Agent

1. Verify `.agentic/agents/{name}/IDENTITY.md` exists. If not, stop and report.
2. Read `IDENTITY.md` and extract:
   - **Role** field: used as the `description` frontmatter value
   - **Activation** field: appended as a trigger clause to the description
   - **Scope** field: informs tool selection via [assets/tool-selection.md](assets/tool-selection.md)
3. Generate `.gemini/agents/{name}.md` using [assets/agent.template.md](assets/agent.template.md)
4. Create `.gemini/agents/` if it does not exist.
5. Confirm the output path.
6. Remind the operator that custom agents are an experimental feature in Gemini CLI and must be enabled in `settings.json` via `"experimental": { "enableAgents": true }`.

## Process: Skill

1. Verify `.agentic/skills/{name}/SKILL.md` exists. If not, stop and report.
2. Read `SKILL.md` and extract the `name` and `description` frontmatter fields.
3. Generate `.gemini/skills/{name}/SKILL.md` using [assets/skill.template.md](assets/skill.template.md)
4. Create `.gemini/skills/{name}/` if it does not exist.
5. Confirm the output path.

## Process: Workflow

1. Verify `.agentic/workflows/{name}/WORKFLOW.md` exists. If not, stop and report.
2. Read `WORKFLOW.md` and extract the `name`, `description`, and `invocation` frontmatter fields.
3. Generate `.gemini/commands/{name}.toml` using [assets/command.template.toml](assets/command.template.toml)
4. Create `.gemini/commands/` if it does not exist.
5. Confirm the output path.

## Rules

- Never copy or paraphrase content from source files into generated file bodies. Always reference back.
- Do not add tools that are not justified by the source artifact's Scope or description. Use [assets/tool-selection.md](assets/tool-selection.md) strictly.
- The `description` frontmatter value must be derived from the source field, not invented.
- The `compatibility` frontmatter value is always `Gemini CLI` regardless of the artifact's domain.
- If a Scope explicitly states no writes or execution, omit all write and execute tools.
