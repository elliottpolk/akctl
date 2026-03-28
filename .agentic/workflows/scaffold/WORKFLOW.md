---
name: scaffold
description: >-
  Materializes the Project Structure from a confirmed spec into real files and
  directories on the feature branch. Reads the spec, confirms the scaffold plan
  with the user, executes, and commits. Run after /spec, before /plan.
invocation: /scaffold
---

# scaffold

Turns the Project Structure section of a confirmed spec into real files and directories
on the feature branch. Nothing is created without user confirmation of the plan first.

## Inputs

- **Spec file path**: the path to the confirmed spec (e.g. `docs/specs/{slug}.md`).
  If not provided, check `manifest.yml` for `settings.target_dir` and list available
  spec files there. Ask the user to pick one.

## Phase 1: Read and Validate the Spec

1. Read the spec file at the provided path.
2. Check the frontmatter `status` field:
   - If `draft`: stop. Report: "This spec is still a draft. Confirm it is ready before scaffolding."
   - If `confirmed` or absent: proceed.
3. Extract from the spec:
   - **Tech Stack**: determines what scaffold tooling or conventions apply
   - **Project Structure**: the list of paths and their descriptions -- this is the scaffold plan
   - **Scope Boundaries MUST always / MUST never**: constraints that apply to every file created
4. If Project Structure is empty or contains only placeholder text, stop and report:
   "Project Structure is not defined in this spec. Add the intended paths before scaffolding."

## Phase 2: Derive and Confirm the Scaffold Plan

1. From the Project Structure entries, derive a concrete action list:
   - Each directory path (`path/`) becomes a `mkdir` with a `.gitkeep` stub unless
     the description implies files within it.
   - Each file path becomes a stub file with a comment header referencing the spec.
   - If the Tech Stack implies a project initializer (e.g. `go mod init`, `npm init`,
     `cargo new`, `rails new`), include it as the first action and note it explicitly.
2. Present the full action list to the user:
   ```
   Scaffold plan for {feature-name}:
   - run: {initializer command}  (if applicable)
   - create: {path}
   - create: {path}
   ...
   ```
3. Ask: "Does this look right? Any paths to add, remove, or change?"
4. Apply any changes. Do not proceed until the user confirms the plan.

## Phase 3: Execute the Scaffold

1. Confirm the working branch is `feature/{slug}`. If not, stop and report the current
   branch. Do not scaffold onto the wrong branch.
2. Apply the Scope Boundaries before creating anything:
   - MUST always constraints apply to every file written
   - MUST never constraints are checked against the plan -- if any action violates one,
     remove it and flag it to the user before proceeding
3. Execute the confirmed plan in order:
   - Run any initializer commands first
   - Create directories and stub files
4. For each stub file, include a single comment at the top:
   `# scaffold: generated from docs/specs/{slug}.md` (adjust comment syntax to the file type)
5. Do not generate implementation. Stub files contain only the scaffold comment and,
   if the file type requires it, a minimal valid structure (e.g. an empty `package main`
   for Go, an empty module export for JS/TS).

## Phase 4: Commit and Handoff

1. Stage all created files.
2. Commit with message: `scaffold: {feature-name}`
3. Confirm: "Scaffold committed on `feature/{slug}`. Ready for `/plan`."
4. Instruct the user to invoke `/plan` with `{spec-path}` as the input brief.
   If the platform supports it, invoke `/plan` directly.

## Rules

- MUST NOT modify or overwrite any file that already exists. Scaffold creates; it does not update.
- MUST NOT generate implementation code. Stubs only.
- MUST confirm the plan with the user before creating any file.
- MUST verify the working branch before executing.
- MUST apply Scope Boundaries from the spec before any action.
- Do not scaffold onto the default branch.
