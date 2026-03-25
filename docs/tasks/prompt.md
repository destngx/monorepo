# Ralph Loop — Instruction Manual

## §0 — Architectural North Star

**Design Philosophy**: Every reusable pattern in one place, zero duplication.

- **Workspace**: Nx Monorepo (`apps/`, `libs/`).
- **Runtime**: `bun` preferred for all commands (`bun install`, `bun nx ...`).
- **Internal Libraries**: Use `@wealth-management/*` aliases. Avoid relative paths between libraries.
- **Imports**: Barrel-only imports where possible.
- **UI Standards**: Use `shadcn/ui` and `Tailwind CSS`. Prioritize `shadcn/ui` for complex components and `Tailwind` for layout/spacing.
- **Error Handling**: Use `AppError` subclasses from `@wealth-management/utils/errors`. NEVER swallow errors.
- **Data Fetching**: Prefer `SWR` for client-side and `getCached`/`setCache` for server-side.

## §1 — Core Loop

FOLLOW THESE 12 STEPS EXACTLY EVERY ITERATION:

1. Read `docs/tasks/prompt.md` (this file) to refresh rules.
2. Read `docs/tasks/PRD.md` to identify the current phase and next task.
3. Read `docs/tasks/friction-log.md` to avoid known pitfalls.
4. Pick the NEXT incomplete task (first ❌ in PRD).
5. Identify the scope (files/libs affected).
6. Research existing patterns in the target component.
7. Execute the task (Code, Refactor).
8. Run local verification (Type-check, Lint, Test).
9. Adversarial Self-Review (See §1.1).
10. Commit changes with a descriptive message.
11. Append results to `docs/tasks/progress.txt`.
12. Update `docs/tasks/PRD.md` (flip ❌ → ✅, update pointer).

**Constraint**: Max 1 task per iteration.

## §1.1 — Adversarial Self-Review

Reread your own work as a hostile reviewer. Run this checklist:

1. Hardcoded text? (Should be in constants or i18n).
2. Duplicate wrappers? (Check if a Layout/Provider already exists).
3. Wrong imports? (Relative path into another lib?).
4. Unused imports/variables?
5. Spelling errors?
6. Missing Error Boundaries or try/catch for async?
7. Missing type safety (any/unknown)?
8. Inconsistent naming?
9. Missing documentation for exports?
10. Over-complicated logic?
11. Circular dependencies?
12. Large files? (Break into smaller components).

Loop up to 3 passes until violations hit zero.

## §3 — Anchored TypeScript Interfaces

```typescript
// From libs/wealth-management/src/services/services/market-data-service.ts

export interface MarketAsset {
  symbol: string;
  name: string;
  market: 'US' | 'VN';
  price: number;
  percentChange: number;
  // ...
}

export interface Technicals {
  cycle: {
    phase: string;
    description: string;
    // ...
  };
  // ...
}
```

## §8 — Friction Log Protocol

When hitting a bug, API mismatch, or environmental issue:

1. Document the issue in `docs/tasks/friction-log.md`.
2. Include the context, error message, and proposed workaround.
3. NEVER assume the next agent knows about this.

## §9 — Component Quick Reference

- Data List -> `InfoCard` or `DataTable`.
- Charting -> `Recharts` or `Three.js` (for visual intelligence).
- Notifications -> `useToast`.
