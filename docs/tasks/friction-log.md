# Friction Log

This file tracks bugs, API mismatches, or environmental issues encountered during development.

---

- **Nx-Go Plugin Friction**: Encountered `received an instance of Array` error during project graph processing with `@nx-go/nx-go` v3.3.1 and Nx v22.6.1. Re-running `nx reset` resolved the issue.
- **Svelte Plugin Friction**: First-party `@nx/svelte` is not available. Community plugin `@nxext/sveltekit` also caused graph issues. Decision: Scaffold manually with `bunx sv create` and add `project.json` for Nx integration.
