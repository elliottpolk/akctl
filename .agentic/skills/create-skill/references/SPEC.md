# agentskills.io Specification Reference

> **Note**: This file is a cached fallback. Agents MUST fetch the live spec at `https://agentskills.io/specification` before creating a skill. Use this file only if that URL is unreachable.

## Directory Structure

A skill is a directory containing, at minimum, a `SKILL.md` file:

```
skill-name/
├── SKILL.md          # Required: metadata + instructions
├── scripts/          # Optional: executable code
├── references/       # Optional: documentation
├── assets/           # Optional: templates, resources
└── ...               # Any additional files or directories
```

## Frontmatter Field Reference

### `name` (required)

- 1-64 characters
- Lowercase letters (`a-z`), numbers (`0-9`), and hyphens (`-`) only
- Must not start or end with a hyphen
- Must not contain consecutive hyphens (`--`)
- Must match the parent directory name exactly

### `description` (required)

- 1-1024 characters
- Must describe both what the skill does and when to use it
- Include specific keywords that help agents identify relevant tasks

### `license` (optional)

- Name of the license or reference to a bundled license file
- Keep it short

### `compatibility` (optional)

- 1-500 characters if provided
- Only include if the skill has specific environment requirements
- Indicate intended product, required system packages, network access needs, etc.

### `metadata` (optional)

- A map from string keys to string values
- Use for additional properties not covered by the spec
- Use reasonably unique key names to avoid conflicts

### `allowed-tools` (optional, experimental)

- Space-delimited list of pre-approved tools
- Support varies between agent implementations

## Progressive Disclosure Model

| Layer | What loads | Size target |
|---|---|---|
| Metadata | `name` and `description` only | ~100 tokens |
| Instructions | Full `SKILL.md` body | Under 5000 tokens / 500 lines |
| Resources | Files in `scripts/`, `references/`, `assets/` | On demand |

Keep `SKILL.md` under 500 lines. Move detailed reference material to separate files.

## Optional Directory Purposes

### `scripts/`

Executable code agents can run. Scripts must be self-contained or clearly document dependencies. Include helpful error messages and handle edge cases gracefully.

### `references/`

Additional documentation agents load when needed:
- `REFERENCE.md` for detailed technical reference
- `FORMS.md` for form templates or structured data formats
- Domain-specific files for specialized content

Keep reference files focused. Agents load them on demand; smaller files consume less context.

### `assets/`

Static resources: templates, images, data files, schemas.

## File Reference Conventions

Use relative paths from the skill root:

```markdown
See [the reference guide](references/REFERENCE.md) for details.
Run: scripts/extract.py
```

Keep references one level deep. Avoid deeply nested chains.

## Validation

To validate a skill against the spec:

```bash
skills-ref validate ./my-skill
```

Checks:
- `SKILL.md` frontmatter is valid
- All naming conventions are followed
