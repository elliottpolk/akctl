---
name: spec
description: >-
  Collaborative, interactive spec creation. Guides user and agent through defining a
  feature spec one section at a time, then hands off to /plan to produce a plan.
  Bridgeable as a /spec prompt in GitHub Copilot or Claude.
invocation: /spec
---

# spec

An interactive workflow for producing a structured feature spec. The agent leads the
conversation, asks questions one section at a time, builds the spec collaboratively
with the user, then hands off to `/plan`.

## Configuration

Target directory is read from `manifest.yml` under this workflow's `settings.target_dir`.
Default: `docs/specs/`.

## Phase 1: Understand the Feature

Ask the user one section at a time. Do not present all questions at once. Reflect their
answer back before moving to the next section.

1. **Name**: What should this feature or task be called? (becomes the filename slug)
2. **Objective**: What problem does this solve, or what outcome does it enable? One to two sentences.
3. **Tech stack**: What languages, frameworks, services, or external systems are involved?
4. **Scope boundaries** (NEED/WANT/MAY applied to agent behavior):
   - What MUST the agent always do? (non-negotiable: safety, security, required patterns)
   - What MUST the agent ask about before doing? (risky or irreversible changes, new deps)
   - What MUST the agent never do? (off-limits actions; things that require human sign-off)
   - If the user skips this section, do not proceed. Prompt once more: "Boundaries are required before an agent can safely execute this. Even a short list helps."
5. **Acceptance criteria**: For each measurable outcome, capture it as Given/When/Then.
   Offer to derive these from the objective if the user is unsure where to start.
6. **Open questions**: What is unresolved, unclear, or could block execution?
   This section is optional. If the user has none, record it as empty.

## Phase 2: Build the Spec

Once all required sections are answered:

1. Read `manifest.yml`. Find this workflow's `settings.target_dir`. Fall back to `docs/specs/` if not set.
2. Slugify the feature name: lowercase, hyphens only, no spaces or special characters.
3. Derive the branch name: `feature/{slug}`.
4. Derive the output path: `{target_dir}/{slug}.md`
5. Write the spec file using the template at `assets/spec.template.md`. Set `name` to
   the feature name and `branch` to `feature/{slug}` in the frontmatter.
   Set `scaffold: true` if the Project Structure section implies creating new files or
   directories that do not yet exist; set `scaffold: false` otherwise.
6. Show the user the generated file path, branch name, and a one-paragraph summary of
   what was captured.
7. Ask: "Does this look right, or do you want to adjust anything?"
8. Apply any changes. Confirm the spec is final before proceeding.

## Phase 2b: Create the Branch and Commit the Spec

Once the spec is confirmed as final:

1. Create the branch `feature/{slug}` from the current default branch.
   - If the branch already exists, stop and ask the user how to proceed.
2. Commit the spec file as the first and only commit on the new branch.
   - Commit message: `spec: add {feature-name} spec`
3. The spec file MUST NOT be committed to the default branch before the feature branch.
   The spec travels with the work and merges via PR.
4. Confirm: "Branch `feature/{slug}` created. Spec committed as first commit."

## Phase 3: Handoff

Once the spec is confirmed as final:

1. Read the `scaffold` field from the spec frontmatter.
   - If `true`: state "Spec is final at `{target_dir}/{slug}.md`. Run `/scaffold` next to
     materialize the project structure, then `/plan` to generate the execution plan."
   - If `false`: state "Spec is final at `{target_dir}/{slug}.md`. Ready for `/plan`."
2. The plan output MUST land at `{target_dir}/{slug}.plan.md`.
3. If the current platform supports it (GitHub Copilot, Claude), invoke the next workflow
   directly rather than asking the user to do it manually.

## Rules

- NEVER fill in a spec section without user input. Every field requires the user to
  provide or confirm the content. The agent MAY suggest or offer to draft a section,
  but the user MUST approve it before it is written.
- MUST complete Phase 1 fully before writing any file.
- Boundaries MUST be explicit and non-empty before Phase 2 begins.
- Do not add a Plan section to the spec. `/plan` generates the plan separately.
- Avoid "should" in the generated spec. Use NEED/MUST, WANT/PREFER, or MAY/CAN.
