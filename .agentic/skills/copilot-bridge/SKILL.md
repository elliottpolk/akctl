---
name: copilot-bridge
description: >-
  Bridges .agentic agents, skills, and workflows into GitHub Copilot. Produces thin
  platform-specific wrapper files that reference canonical sources rather than
  duplicating them. Use when making any .agentic component available in GitHub Copilot Chat.
compatibility: GitHub Copilot in VS Code
---

# copilot-bridge

Produces GitHub Copilot wrapper files from components defined in `.agentic/`. No content
is copied from source files — all generated files reference back to their canonical source.

## Artifact Types and Output Paths

| Input | Source | Output |
|---|---|---|
| Agent | `.agentic/agents/{name}/IDENTITY.md` | `.github/agents/{name}.agent.md` |
| Skill | `.agentic/skills/{name}/SKILL.md` | `.github/skills/{name}/SKILL.md` |
| Workflow | `.agentic/workflows/{name}/WORKFLOW.md` | `.github/prompts/{name}.prompt.md` |

## Inputs

- **Type**: one of `agent`, `skill`, or `workflow`
- **Name**: the directory name under the relevant `.agentic/` subdirectory

## Process: Agent

1. Verify `.agentic/agents/{name}/IDENTITY.md` exists. If not, stop and report.
2. Read `IDENTITY.md` and extract:
   - **Role** field: used as the `description` frontmatter value
   - **Activation** field: used as the `argument-hint` frontmatter value
   - **Scope** field: informs tool selection via [assets/tool-selection.md](assets/tool-selection.md)
3. Generate `.github/agents/{name}.agent.md` using [assets/agent.template.md](assets/agent.template.md)
4. Create `.github/agents/` if it does not exist.
5. Confirm the output path.

## Process: Skill

1. Verify `.agentic/skills/{name}/SKILL.md` exists. If not, stop and report.
2. Read `SKILL.md` and extract the `name` and `description` frontmatter fields.
3. Generate `.github/skills/{name}/SKILL.md` using [assets/skill.template.md](assets/skill.template.md)
4. Create `.github/skills/{name}/` if it does not exist.
5. Confirm the output path.

## Process: Workflow

1. Verify `.agentic/workflows/{name}/WORKFLOW.md` exists. If not, stop and report.
2. Read `WORKFLOW.md` and extract the `name`, `description`, and `invocation` frontmatter fields.
3. Generate `.github/prompts/{name}.prompt.md` using [assets/prompt.template.md](assets/prompt.template.md)
   - Check the workflow's frontmatter for an `agent:` field. If present, use its value.
   - If absent, default to `agent: agent`.
4. Create `.github/prompts/` if it does not exist.
5. Confirm the output path.

## Rules

- Never copy or paraphrase content from source files into generated file bodies. Always reference back.
- Do not add tools that are not justified by the source artifact's Scope or description.
- The `description` frontmatter value must be derived from the source field, not invented.
- If a Scope explicitly states no writes or execution, omit all edit and terminal tools.
