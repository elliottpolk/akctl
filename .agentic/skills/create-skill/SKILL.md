---
name: create-skill
description: >-
  Create a new agent skill following the agentskills.io specification. Use this skill
  when asked to create a new skill, add a skill to this workspace, or package agent
  instructions as a reusable, shareable skill. Handles directory structure, SKILL.md
  frontmatter, body content, and optional supporting files.
license: see ../../../LICENSE
compatibility: >-
  Designed for use in the elliottpolk/docs knowledge base. Skills are created in
  .agentic/skills/ and symlinked via .github/skills/. No external network access
  required at skill-creation time.
metadata:
  spec-url: https://agentskills.io/specification
  spec-version: "1.0"
---

# Create Skill

You are creating a new agent skill following the [agentskills.io specification](https://agentskills.io/specification). The result is a directory in `.agentic/skills/` with a valid `SKILL.md` file and any supporting assets.

## Before You Start

1. **Fetch the live spec** from `https://agentskills.io/specification` before proceeding. Use the fetched content as the authoritative source for field constraints, naming rules, and directory structure. The spec may have changed since this skill was written; the live version wins.
   - If the URL is unreachable, fall back to [references/SPEC.md](references/SPEC.md), but note it may be stale.
2. Confirm the skill name with the user if not already stated. The name MUST be lowercase, hyphenated, and match the directory name exactly. No consecutive hyphens.
3. Confirm the skill's purpose: what it does and when an agent activates it.
4. Ask whether supporting files are needed (scripts, references, assets) or if `SKILL.md` alone is sufficient.

Do not create anything until you have a name, a clear description, and the current spec in hand.

## Step 1: Plan the Structure

Decide what the skill needs:

| Component | Include when... |
|---|---|
| `SKILL.md` | Always. Required. |
| `scripts/` | The skill runs executable code (Python, Bash, etc.) |
| `references/` | Detailed reference material would bloat SKILL.md beyond ~500 lines |
| `assets/` | The skill uses templates, data files, or static resources |

Skills in this workspace follow the progressive disclosure model: keep `SKILL.md` under 500 lines. Move large reference content to `references/`.

## Step 2: Create the Directory

```
.agentic/skills/{skill-name}/
```

The directory name MUST exactly match the `name` field in `SKILL.md`.

## Step 3: Write SKILL.md

### Frontmatter

Required fields:

```yaml
---
name: {skill-name}
description: >-
  {What the skill does. When an agent activates it. Keywords that help agents
  identify relevant tasks. Max 1024 characters.}
---
```

Optional fields to add when relevant:

```yaml
license: see ../../../LICENSE          # Use this for workspace skills
compatibility: >-                      # Only if environment requirements exist
  {Relevant context: intended products, system packages, required paths}
metadata:                              # Arbitrary key-value for extra context
  {key}: {value}
allowed-tools: {space-delimited list}  # Experimental; pre-approves tool calls
```

### Frontmatter Rules

- `name`: 1-64 chars. Lowercase letters, numbers, hyphens only. No leading/trailing/consecutive hyphens. Must match the directory name.
- `description`: 1-1024 chars. Non-empty. Describe both what it does and when to use it. Include keywords.
- `license`: use `see ../../../LICENSE` for workspace skills.
- `compatibility`: only include if the skill has specific environment requirements.

### Body Content

Write clear, agent-executable instructions. Recommended sections:

- Step-by-step instructions
- Examples of inputs and outputs
- Common edge cases
- Acceptance criteria or definition of done

Keep the body under 500 lines. Reference detail files if content would exceed that.

### File References

When referencing other files, use relative paths from the skill root:

```markdown
See [the reference guide](references/REFERENCE.md) for details.
Run: scripts/my-script.py
```

Keep references one level deep. No deeply nested chains.

## Step 4: Add Supporting Files (If Needed)

### scripts/

- One script per discrete operation
- Self-contained or clearly document dependencies
- Include helpful error messages
- Use clear names: `check-dashes.py`, `validate.sh`, not `run.py`

### references/

- Keep individual files focused on one topic
- Agents load these on demand; smaller files consume less context
- `REFERENCE.md` for general deep reference
- Domain-specific files for specialized content

### assets/

- Templates, data files, schemas
- Static resources the skill needs at runtime

## Step 5: Validate

Check the skill before marking done:

- [ ] Directory name matches `name` frontmatter field exactly
- [ ] `name` is lowercase, hyphenated, no leading/trailing/consecutive hyphens
- [ ] `description` is non-empty and explains both what and when
- [ ] `SKILL.md` body is clear and executable by an agent reading it cold
- [ ] No em dashes (U+2014) or en dashes (U+2013) in any file
- [ ] No "should" in any prose (use declarative language or MUST/WANT/MAY)
- [ ] File references use relative paths from the skill root
- [ ] `SKILL.md` is under 500 lines (move excess to `references/` if not)

## Workspace Notes

- Skills live in `.agentic/skills/` (source of truth)
- `.github/skills/` is a symlink to `.agentic/skills/`; do not create files there directly
- Existing skills (`check-em-en-dash`, `check-should`, `toph-review-initiative`) are examples of the pattern
- Follow language discipline rules that apply throughout this workspace: no em/en dashes, no "should"

## Common Mistakes

- **Name mismatch**: directory is `my-skill` but frontmatter says `name: myskill`. These MUST match.
- **Consecutive hyphens**: `my--skill` is invalid.
- **Bloated SKILL.md**: instructions that run 800 lines belong in `references/REFERENCE.md`, not inline.
- **Vague description**: "Helps with tasks" tells an agent nothing. Name the domain, the trigger, and the output.

See [references/SPEC.md](references/SPEC.md) for a cached spec summary (fallback only; fetch the live spec first).
