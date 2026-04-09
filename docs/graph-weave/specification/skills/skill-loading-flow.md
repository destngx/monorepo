## 1. Objective

- What: Show how GraphWeave loads skills in three levels.
- Why: Keep orchestrator context small while still allowing full capability expansion on demand.
- Who: Runtime engineers and AI workflow authors.

## Traceability

- FR-SKILL-001 [MVP,FULL]: Level 1 frontmatter must always be present.
- FR-SKILL-002 [MVP,FULL]: Level 2 skill bodies must be loaded dynamically at runtime.
- FR-SKILL-003 [MVP,FULL]: Missing Level 2 bodies or required Level 3 linked files must fail the node.

## 2. Scope

- In scope: level-1 summaries, level-2 bodies, level-3 linked files, lazy loading, and failure handling.
- Out of scope: subagent tool execution internals and provider-specific parsing.

## 3. Specification

- Level 1 frontmatter must always be available to the orchestrator.
- The loading model is three-level: Level 1 frontmatter, Level 2 body, and Level 3 linked files.
- Mapping: Level 1 = discovery metadata, Level 2 = body instructions, Level 3 = linked assets.
- Level 1 metadata must stay minimal and focus on name, description, and trigger cues.
- Skill discovery must come from folder layout plus frontmatter metadata, not from runtime code scanning.
- Level 2 skill bodies must be loaded only when the orchestrator selects a specific skill.
- Level 3 linked files must load only when the skill execution needs them.
- Missing Level 2 bodies or required Level 3 linked files must fail the node.
- The skills model is a hard requirement, not an optimization.
- Inputs to skills may come from folder-based docs or MCP tools, but runtime loading is mandatory.
- NFR: loading should minimize prompt bloat and preserve routing responsiveness.

## 4. Technical Plan

- Load Level 1 frontmatter first and inject it into the orchestrator prompt.
- Use Redis as the lookup cache for discovering which skills exist and whether they are relevant.
- Fetch Level 2 bodies only for selected skills.
- Open Level 3 linked files only when the execution path needs them.
- Track loaded skill artifacts in active context state for agent execution.
- Keep the loading flow explicit enough to explain why a skill was or was not expanded.
- Maintain a clear convention for naming skill-loading events in streams.
- The Skills layer owns lookup metadata, the Runtime layer owns pre-loading, and Redis stores tenant-scoped cached lookup entries; the interpreter receives pre-loaded skills and does not perform registry discovery itself.
- On cache miss, the Runtime layer reloads from folder/frontmatter source of truth, updates the Redis lookup entry, and then continues loading.
- Skill cache invalidation is explicit and API-driven so external edits can be reflected without restarting the runtime.
- Skill invalidation should be a dedicated API action that accepts tenant scope, skill identifier, and reason, then deletes the cached lookup entry so the next load rebuilds from source of truth.
- Skill version changes should be treated as cache invalidation triggers, not as ad hoc runtime mutations.
- Skill cache keys must include a version segment, and the default lookup target when no version is specified is `latest`.

## 5. Tasks

- [ ] Load level-1 frontmatter into Redis-backed state.
- [ ] Fetch level-2 bodies lazily when requested.
- [ ] Open level-3 linked files only on demand.
- [ ] Fail the node when a required artifact is missing.
- [ ] Add naming guidelines for skill-load events and failure events.

## 6. Verification

- Given a workflow start, when it begins, then Level 1 frontmatter must be available before routing.
- Given the orchestrator selects a skill, when the loader runs, then only that skill's Level 2 body should load.
- Given a skill needs supporting files, when execution requests them, then only the needed Level 3 linked files should open.
- Given a missing required artifact, when the loader fails to find it, then the node should fail.
- Given a skill activation, when the runtime expands context, then it must be clear why the expansion occurred.

```mermaid
flowchart TD
    A[Start workflow] --> B[Load Level 1 frontmatter from Redis]
    B --> C[Inject frontmatter into orchestrator prompt]
    C --> D{Need a specific skill?}
    D -->|No| E[Continue with Level 1 only]
    D -->|Yes| F[Load Level 2 skill body]
    F --> G{Need linked files?}
    G -->|Yes| H[Open Level 3 linked files]
    G -->|No| I[Execute with body only]
    H --> J{All required artifacts present?}
    I --> K[Return to orchestrator decision]
    J -->|Yes| L[Execute agent with full skill context]
    J -->|No| M[Fail node and surface error]
    E --> K
    L --> K
    M --> K
```
