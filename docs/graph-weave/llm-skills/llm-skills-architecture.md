# LLM Skills Architecture

## Skill Layout

A skill is a folder containing:

- `SKILL.md` (required): Markdown instructions with YAML frontmatter
- `scripts/` (optional): executable helpers
- `references/` (optional): docs loaded on demand
- `assets/` (optional): templates or other output resources

## Loading Model

Skills use progressive disclosure:

1. Frontmatter is always loaded into the system prompt so the agent knows when to use the skill.
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
