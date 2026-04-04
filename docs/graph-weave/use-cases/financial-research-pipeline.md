# Use case: Financial Research Pipeline Stagnation Scenario

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
