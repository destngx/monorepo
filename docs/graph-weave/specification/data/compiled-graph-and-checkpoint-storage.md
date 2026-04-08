# Compiled Graph and Checkpoint Storage

## 1. Objective

- What: Define how GraphWeave stores compiled graphs and execution checkpoints.
- Why: Keep reusable graph caches separate from live resume state.
- Who: Runtime engineers and platform operators.

## Traceability

- FR-DATA-003: Compiled graphs and checkpoints must be independently addressable.
- FR-DATA-004: Cached runtime artifacts must remain distinct from live execution state.

## 2. Scope

- In scope: compiled graph cache intent, checkpoint state, TTL separation, and resume boundaries.
- Out of scope: serialization format details and retention policy tuning.

## 3. Specification

- Compiled graphs must be stored separately from checkpoints.
- Checkpoints must capture run_id, version, last_node, timestamp, and execution state.
- Checkpoints must include interpreter state, event history, and execution context needed to resume.
- Compiled graphs are cacheable; checkpoints are resume state.
- TTL for compiled graphs and checkpoints may differ, but the intent must be explicit.
- Resume must restore the last known node and continue the workflow safely.
- Runtime state that cannot be reconstructed on resume must be marked ephemeral.

## 4. Technical Plan

- Keep compiled graph cache keys separate from checkpoint keys.
- Treat compiled graphs as reusable artifacts that can be re-fetched if cache entries expire.
- Keep checkpoint records minimal but complete enough to resume execution.
- Preserve compatibility when the namespace shape changes.
- Make it obvious which runtime artifacts are live state versus cached execution artifacts.

## 5. Tasks

- [ ] Define checkpoint fields and what they capture.
- [ ] Keep compiled graph cache separate from checkpoint storage.
- [ ] Document resume behavior from the last saved node.
- [ ] Make TTL intent explicit for both storage families.

## 6. Verification

- Given a run checkpoint, when it is restored, then execution resumes from the last saved node.
- Given a compiled graph cache entry, when it expires, then it can be regenerated or re-fetched without losing checkpoint state.
- Given both cached graphs and checkpoints exist, when storage is inspected, then they must be clearly separable.
