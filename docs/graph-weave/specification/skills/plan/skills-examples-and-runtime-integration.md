# Skills Examples and Runtime Integration

Purpose: record how examples and supporting files relate to runtime loading without becoming part of the core contract.

## Scope

- examples folder usage
- optional scripts and references
- runtime integration boundaries
- prompt/context safety notes

## References and context

- `[[../examples/README]]` says the examples folder is for short, concrete skill examples that link back to the main skill docs. This reference matters because examples are support material, not the contract itself.
- `[[../README]]` keeps the scope at the skills layer. This reference is included so the plan does not drift into runtime execution or workflow routing.

## Decisions

- Keep examples short and concrete.
- Allow supporting references to stay external until needed.
- Preserve a clean boundary between the skill body and optional assets.
- Keep sensitive or noisy context out of discovery metadata.

## Definition of done

- Example usage is documented as support material.
- Runtime integration remains tied to the skill contract, not ad hoc files.
- The plan explains why examples exist and how they are consumed.

## Next links

- `[[skill-loading-and-packaging]]`
- `[[skill-registry-and-metadata-contracts]]`
