# LLM Skills Architecture

## 1. Objective

- What: Define how GraphWeave packages and loads skills.
- Why: Make skills reusable, discoverable, and context-efficient.
- Who: AI workflow authors and runtime engineers.

## Traceability

- FR-SKILL-010: Skills must use progressive disclosure.
- FR-SKILL-011: Skill packages must support multiple skills simultaneously.
- FR-SKILL-012: Skill packaging must work with MCP tooling and dynamic runtime loading.

## 2. Scope

- In scope: skill folder layout, frontmatter, body loading, and linked reference files.
- Out of scope: provider implementation details inside individual skills.

## 3. Specification

- Skills must expose concise frontmatter for discovery and detailed bodies for on-demand loading.
- The system must support multiple skills at once.
- Skills must be written so they work alongside MCP tooling, not instead of it.
- The folder format is illustrative, but the loading contract is mandatory.
- The docs should distinguish discovery metadata from the full body contract.
- Discovery metadata must be a minimal JSON-compatible summary used for Tier 1 loading.
- NFR: skills should be concise enough to keep prompts compact but rich enough to be actionable.

## 4. Technical Plan

- Discover skills from folder structure and frontmatter metadata.
- Load the full body only when the skill is selected.
- Keep supporting references external until needed.
- Keep skill packaging compatible with both folder-based discovery and registry-based loading.
- Preserve a clear boundary between metadata, instructions, and supporting examples.
- Keep the runtime contract aligned with the prompt-driven agent_node model, not skill_call routing.

## 5. Tasks

- [ ] Define the skill folder contract.
- [ ] Keep frontmatter concise and discovery-friendly.
- [ ] Preserve body sections for workflow, examples, troubleshooting, and references.
- [ ] Add examples for discovery metadata and full-body loading.

## 6. Verification

- Given a skill folder, when it is discovered, then the frontmatter should be enough to decide whether it is relevant.
- Given a selected skill, when it is activated, then the full body should load on demand.
- Given multiple skills are available, when the agent needs them, then they should be able to work together.
- Given a skill package, when it is consumed by the runtime, then discovery and load-time behavior must remain distinct.

## Skill Layout

A skill is a folder containing:

- `SKILL.md` (required): Markdown instructions with YAML frontmatter
- `scripts/` (optional): executable helpers
- `references/` (optional): docs loaded on demand
- `assets/` (optional): templates or other output resources

## Loading Model

Skills use progressive disclosure:

1. Frontmatter is always loaded into the system prompt as the Tier 1 summary so the agent knows when to use the skill.
2. The `SKILL.md` body loads only when the skill is relevant.
3. Linked files stay external until needed.

This keeps context small while preserving detailed workflow guidance.

In practice, the frontmatter should describe what the skill does and when to use it, while the body should carry the steps, examples, troubleshooting, and references.

## Why It Helps

- Lower token usage
- Consistent tool use
- Reusable workflows
- Multiple skills can work together

## MCP + Skills

MCP provides tools; skills provide the operating playbook. Together they let agents apply the right tool with the right procedure, instead of rediscovering the workflow every time.

Think of MCP as the kitchen equipment and skills as the recipes: the tool exists, but the skill tells the agent how to use it safely and consistently.

## Authoring Rules

- Keep the description short and specific
- Use kebab-case skill names
- Include workflow, examples, troubleshooting, and references in the body
- Keep sensitive tags out of frontmatter to reduce prompt-injection risk
