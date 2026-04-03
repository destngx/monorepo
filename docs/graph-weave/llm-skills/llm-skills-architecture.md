# LLM Skills Architecture

## What is Skill?

A skill is a folder containing:
- `SKILL.md` (required): Instructions in Markdown with YAML frontmatter
- `scripts/` (optional): Executable code (Python, Bash, etc.)
- `references/` (optional): Documentation loaded as needed
- `assets/` (optional): Templates, fonts, icons used in output

Skills use a three-level system:
- First level (YAML frontmatter): Always loaded in Agent's system prompt. Provides just enough information for Agent to know when each skill should be used without loading all of it into context.
- Second level (SKILL.md body): Loaded when Agent thinks the skill is relevant to the current task. Contains the full instructions and guidance.
- Third level (Linked files): Additional files bundled within the skill directory that Agent can choose to navigate and discover only as needed.

Benefit:
- This progressive disclosure minimizes token usage while maintaining specialized expertise.
- Pre-built workflows activate automatically when needed
- Consistent, reliable tool usage
- Best practices embedded in every interaction
- Lower learning curve for your integration

Agents can load multiple skills simultaneously. Your skill should work well alongside others, not assume it's the only capability available

### For MCP Builders: Skills + Connectors

If you already have a working MCP server, you've done the hard part. Skills are the knowledge layer on top - capturing the workflows and best practices you already know, so Agents can apply them consistently.

> The kitchen analogy:
MCP provides the professional kitchen: access to tools, ingredients, and equipment. Skills provide the recipes: step-by-step instructions on how to create something valuable. Together, they enable users to accomplish complex tasks without needing to figure out every step themselves. 

## How Skills are used

Scan the directory and find SKILL.md files and parse their frontmatter
  - The description must have: What the skill does? When to use it, trigger conditions? Short lenght (<1024 characters)
  - The function name is skill name, MUST be ke-bab case
  - Convert each Skill into LLM skill function (https://www.google.com/url?sa=E&q=https%3A%2F%2Fplatform.openai.com%2Fdocs%2Fguides%2Ffunction-calling)

When LLM decided to use a skill:
  - Only until this time, we load all the content of the skill, body should has structure:
    - Instruction/Workflow: Step by step
    - Examples: Use cases details of what good output.
    - Troubleshooting: React when failed
    - References/Scripts
    - Script can be triggered by path, but MUST not be loaded to save context memory.
  - Important notes should be highlighted using `## Important` or `## Critical` note.

Security notes:
  - prompt injection: never has XML tag inside front matter
