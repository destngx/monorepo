# GW-RUNTIME-002: Define circuit breaker and stagnation detection rules

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : Critical
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime guardrails and safety
- **Feature** : Define execution safeguards to prevent infinite loops and resource exhaustion
- **Rationale** : Safety boundaries must be locked to prevent production incidents and ensure deterministic guardrail behavior

---

### Input

- **Data / Files** : `[[../specification/runtime/circuit-breaker.md]]`, `[[../specification/runtime/stagnation-detection-logic.md]]`, `[[../specification/runtime/plan/guardrails-circuit-breakers-and-stagnation.md]]`, `[[../specification/runtime/plan/README.md]]`
- **Dependencies** : GW-RUNTIME-001 (interpreter interface), GW-ARCH-003 (request lifecycle)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define stagnation detection rules (what constitutes "stuck")
  - Define circuit breaker triggers (what stops execution)
  - Define timeout and iteration limits
  - Document watchdog signals (how guardrails communicate to outer runtime)
  - Document kill-switch flow (immediate termination semantics)
  - Map guardrails to Redis state (control flags, counters)

- **Out of Scope**:
  - Implementation of watchdog thread
  - Redis key naming (see GW-DATA-001)
  - User-facing error messages
  - Recovery or retry logic after circuit break

- **Max Increment**: Complete guardrails specification

---

### Approach

1. Synthesize circuit-breaker, stagnation-detection-logic, and plan docs
2. Define stagnation signals: repeated node visits, timeout, iteration cap
3. Define circuit breaker state machine (armed, tripped, recovering)
4. Document kill-switch semantics (immediate stop vs. graceful drain)
5. Define watchdog communication protocol (events, signals, timing)

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/circuit-breaker.md` — Update with complete circuit breaker state machine
- `docs/graph-weave/specification/runtime/stagnation-detection-logic.md` — Confirm stagnation rules and thresholds
- `docs/graph-weave/specification/runtime/plan/guardrails-circuit-breakers-and-stagnation.md` — Ensure decisions reflected

---

### Expected Output

- **Deliverable** : Circuit breaker and stagnation detection specification
- **Format** : Markdown with state diagrams and rule tables
- **Example** :

```
Stagnation Rule | Trigger | Action | Threshold
Repeated node   | Same node 5 times | Emit stagnation event | iteration_limit=5
Timeout         | Wall clock > limit | Emit timeout event | max_duration_sec=300
Iteration       | Step count > limit | Emit iteration event | max_steps=1000

Circuit State | Condition | Watchdog Action | Recovery
Armed         | Running   | Monitor stagnation | None
Tripped       | Stagnation detected | Emit kill event | Can be reset
Recovering    | Kill in progress | Drain active state | Timeout or success
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-RUNTIME-002.md]]`

---

### References

- `[[../specification/runtime/circuit-breaker.md]]` — Canonical circuit breaker spec; ensure alignment
- `[[../specification/runtime/stagnation-detection-logic.md]]` — Stagnation rules; must be reflected in spec
- `[[../specification/runtime/plan/guardrails-circuit-breakers-and-stagnation.md]]` — Decisions on guardrails; ensure consistency
- `[[../specification/runtime/request-lifecycle.md]]` — Guardrails sit outside interpreter; cross-reference integration point
