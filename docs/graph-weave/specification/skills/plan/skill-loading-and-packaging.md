# Skill Loading and Packaging

Purpose: record the three-level progressive-disclosure loading model so skill context stays small until the runtime needs it.

## Scope

- Level 1 frontmatter
- Level 2 body loading
- Level 3 linked-file loading
- lazy loading and failure handling
- MCP/tool boundary compatibility

## References and context

- `[[../skill-loading-flow]]` defines the three-level loading path and the failure rule when required skill artifacts are missing. This reference anchors the plan to the runtime behavior that must remain stable.
- `[[../llm-skills-architecture]]` defines progressive disclosure, skill packaging, and the separation between discovery metadata, body instructions, and linked files. This reference matters because it is the source of truth for the packaging contract.

## Decisions

- Always load Level 1 frontmatter.
- Load Level 2 only when a specific skill is selected.
- Open Level 3 linked files only when they are required.
- Fail the node if a required artifact is missing.
- Keep the packaging contract compatible with MCP tooling.

## Definition of done

- The three-level loading model is documented.
- The lazy-load failure rule is explicit.
- The plan explains why the runtime sees frontmatter first, body second, and linked files last.

## Next links

- `[[skill-registry-and-metadata-contracts]]`
- `[[skills-examples-and-runtime-integration]]`
