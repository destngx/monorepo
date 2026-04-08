# Skill Registry and Metadata Contracts

Purpose: record the folder layout and discovery metadata that let the runtime find and load skills safely.

## Scope

- skill folder structure
- frontmatter metadata
- discovery vs load-time behavior
- reference and asset boundaries

## References and context

- `[[../llm-skills-architecture]]` defines the folder contract, frontmatter rules, and authoring guidelines. This reference is included because the plan must preserve discovery-friendly metadata and clear body structure.
- `[[../examples/README]]` describes the purpose of example skill docs and registry behavior. This reference matters because examples show how the metadata contract is consumed.

## Decisions

- Keep discovery metadata concise and JSON-compatible.
- Treat body content as load-on-demand guidance.
- Preserve a clear distinction between metadata, workflow steps, examples, references, and linked files.
- Keep folder structure predictable for agent path finding.
- Make the registry index queryable by skill name, category, and version.
- Treat the registry as metadata-only; execution code stays in the skill body.

## Definition of done

- The registry and metadata contract are documented in one place.
- Discovery behavior is distinct from load-time behavior.
- The plan shows how the folder structure supports runtime lookup and linked-file loading.

## Next links

- `[[skill-loading-and-packaging]]`
- `[[skills-examples-and-runtime-integration]]`
