# Use case: Financial Research Pipeline Stagnation Scenario

## 1. Objective

- What: Research earnings, transcripts, and internal performance data without getting trapped in loops.
- Why: Give analysts partial but useful answers instead of infinite retries.
- Who: Hedge fund research desks.

## 2. Scope

- In scope: web search, SQL lookup, synthesis, stagnation detection, and partial-answer recovery.
- Out of scope: manual research workflows and repeated retry loops without stop conditions.

## 3. Specification

- The workflow must alternate between search and SQL where appropriate.
- Repeated directives or objectives must be detected with a sliding window.
- If the same intent repeats three times, the workflow must exit safely and return partial results.

## 4. Technical Plan

- Use a web search agent for filings and transcripts.
- Use a SQL agent for internal performance data.
- Persist recent orchestrator outputs for stagnation comparison.

## 5. Tasks

- [ ] Fetch research sources and SQL-backed metrics.
- [ ] Detect repeated intent or objective patterns.
- [ ] Return the last successful summary on stagnation.

## 6. Verification

- Given repeated JOIN failures, when the same intent repeats, then the workflow must stop rather than loop forever.
- Given a partial result, when stagnation is detected, then the workflow must return it with a stagnation flag.
- Given a normal research run, when the data is available, then it should complete without fallback.

Tenant: hedge fund research desk
Trigger: analyst asks for earnings and SQL-backed performance synthesis
Workflow: `quant-research:v3.0.0`

Execution flow:

- Orchestrator routes to a web search agent for filings and transcripts
- Orchestrator routes to a SQL agent for internal performance data
- Results are summarized and fed back into the next routing pass
- A sliding-window stagnation detector watches for repeated directives or objectives
- If the same intent repeats three times, the graph exits with a partial answer

Example path:

- Web search pulls Q3 transcripts
- SQL agent queries the warehouse for historical performance
- If the SQL agent keeps producing the same failed JOIN intent, the graph stops rather than looping forever

Recovery:

- Return the last successful summary
- Include `stagnation: true` and `partial_results: true`

Edge cases:

- Repeated bad JOIN attempts -> stagnation exit
- Repeated extraction loops -> force exit with warning
