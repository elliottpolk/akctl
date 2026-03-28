# Quality Checklist

Run this before registering a new agent in `manifest.yml`. All items must pass.

## Identity
- [ ] Name is kebab-case and reflects the agent's role without being vague (e.g. `security-reviewer`, not `helper`)
- [ ] Role is one clear sentence
- [ ] Domain is specific enough to distinguish this agent from others
- [ ] Scope states explicitly what the agent does NOT do

## Instructions
- [ ] Operating Principles are specific to this agent, not restatements of `.agentic/core/BEHAVIOR.md`
- [ ] Instructions use imperative language ("Do X", "Never Y"), not suggestions ("should", "try to")
- [ ] Activation condition is defined and unambiguous

## Boundaries
- [ ] No significant overlap with an existing agent, or overlap is explicitly acknowledged and justified
- [ ] The agent does not require platform-specific tooling in its `IDENTITY.md` (platform bridging is handled by skills)

## Structure
- [ ] `IDENTITY.md` created from `assets/IDENTITY.template.md`
- [ ] `notes/` directory exists
- [ ] `memories/` directory exists
- [ ] `assets/` and `references/` added only if needed

## Registration
- [ ] Agent added to `agents:` list in `manifest.yml` with correct path
