# Requirements Gathering

Use this before designing a new agent. All NEED items must be answered before proceeding to Draft.

## NEED (required before proceeding)

- What is the agent's primary role? (one sentence)
- What domain or subject matter does it operate in?
- What are the 2-3 core tasks it will handle?
- What does it explicitly NOT do? (scope boundary)
- Does this role overlap with any existing agent in `manifest.yml`? If yes, is the distinction clear enough to justify a separate agent?

## WANT (capture if available)

- Will this agent operate standalone, or as part of a workflow sequence?
- Are there specific tools, references, or templates it needs in `assets/` or `references/`?
- Are there edge cases or failure modes to encode in Operating Principles?

## MAY (optional, low priority)

- Who or what triggers this agent? (informs the Activation field)
- Are there examples of good output to include as reference?
