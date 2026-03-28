---
name: claude-bridge
description: >-
  Bridges .agentic agents, skills, and workflows into Claude Code and Cowork. Produces
  thin platform-specific wrapper files that reference canonical sources rather than
  duplicating them. Use when making any .agentic component available in Claude.
compatibility: Claude Code, Cowork
---

# claude-bridge

Produces Claude wrapper files from components defined in `.agentic/`. No content is
copied from source files — all generated files reference back to their canonical source.
The same output works across Claude Code and Cowork; no separate file is needed per surface.

## Artifact Types and Output Paths

| Input | Source | Output |
|---|---|---|
| Agent | `.agentic/agents/{name}/IDENTITY.md` | `.claude/skills/{name}/SKILL.md` |
| Skill | `.agentic/skills/{name}/SKILL.md` | `.claude/skills/{name}/SKILL.md` |
| Workflow | `.agentic/workflows/{name}/WORKFLOW.md` | `.claude/commands/{name}.md` |

## Inputs

- **Type**: one of `agent`, `skill`, or `workflow`
- **Name**: the directory name under the relevant `.agentic/` subdirectory

## Process: Agent

1. Verify `.agentic/agents/{name}/IDENTITY.md` exists. If not, stop and report.
2. Read `IDENTITY.md` and extract:
   - **Role** field: used as the `description` frontmatter value
   - **Activation** field: appended as a trigger clause to the description
   - **Scope** field: informs tool selection via [assets/tool-selection.md](assets/tool-selection.md)
3. Generate `.claude/skills/{name}/SKILL.md` using [assets/agent.template.md](assets/agent.template.md)
4. Create `.claude/skills/{name}/` if it does not exist.
5. Confirm the output path.
6. If `IDENTITY.md` includes a **Model** preference field, note it in the confirmation but do not embed it in the generated file.

## Process: Skill

1. Verify `.agentic/skills/{name}/SKILL.md` exists. If not, stop and report.
2. Read `SKILL.md` and extract the `name` and `description` frontmatter fields.
3. Generate `.claude/skills/{name}/SKILL.md` using [assets/skill.template.md](assets/skill.template.md)
4. Create `.claude/skills/{name}/` if it does not exist.
5. Confirm the output path.

## Process: Workflow

1. Verify `.agentic/workflows/{name}/WORKFLOW.md` exists. If not, stop and report.
2. Read `WORKFLOW.md` and extract the `name`, `description`, and `invocation` frontmatter fields.
3. Generate `.claude/commands/{name}.md` using [assets/command.template.md](assets/command.template.md)
4. Create `.claude/commands/` if it does not exist.
5. Confirm the output path.

## Rules

- Never copy or paraphrase content from source files into generated file bodies. Always reference back.
- Do not add tools that are not justified by the source artifact's Scope or description. Use [assets/tool-selection.md](assets/tool-selection.md) strictly.
- The `description` frontmatter value must be derived from the source field, not invented.
- The `compatibility` frontmatter value is always `Claude Code, Cowork` regardless of the artifact's domain.
- If a Scope explicitly states no writes or execution, omit all write and execute tools.

## Claude API / SDK Usage

For agents specifically: the Claude API requires no bridge file. Pass the contents of
`.agentic/agents/{name}/IDENTITY.md` directly as the `system` parameter. Mention this
to the operator after generating the skill file.
