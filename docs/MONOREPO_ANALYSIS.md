# Monorepo Analysis: AGENTS.md Necessity Scorecard

## Executive Summary

**Monorepo**: `@destnguyxn-monorepo` (Nx + Next.js + Python)  
**Total Projects**: 12 (4 apps + 8 libs)  
**Total Files**: 947 (excluding node_modules, .next, **pycache**)  
**Languages**: TypeScript/React (11), Python (1)

### Recommendation

Create `AGENTS.md` in **exactly 3 directories**:

1. ✅ `apps/wealth-management/` - Complex Next.js with OAuth + AI setup
2. ✅ `apps/vnstock-server/` - Python backend (isolated ecosystem)
3. ✅ `libs/wealth-management/` - 348 TypeScript files (highest code complexity)

---

## Complexity Scorecard (Ranked)

| Rank | Project                                       | Type           | Score | Files | Size    | Language | Needs AGENTS.md?    |
| ---- | --------------------------------------------- | -------------- | ----- | ----- | ------- | -------- | ------------------- |
| 1    | `apps/portfolio-landpage`                     | Next.js App    | 588.9 | 50    | 53.89MB | TS       | ❌ No (asset-heavy) |
| 2    | `libs/wealth-management`                      | TypeScript Lib | 357.6 | 348   | 0.96MB  | TS       | ✅ YES              |
| 3    | `apps/wealth-management`                      | Next.js App    | 174.8 | 164   | 1.08MB  | TS       | ✅ YES              |
| 4    | `apps/vnstock-server`                         | Python App     | 127.6 | 118   | 0.96MB  | Py       | ✅ YES              |
| 5    | `apps/cloudinary-photos-app`                  | Next.js App    | 66.2  | 56    | 1.02MB  | TS       | ❌ No               |
| 6    | `libs/portfolio-landpage/components`          | React Lib      | 30.2  | 30    | 0.02MB  | TS       | ❌ No               |
| 7    | `libs/cloudinary-photos-app/components/ui`    | React Lib      | 27.3  | 27    | 0.03MB  | TS       | ❌ No               |
| 8    | `libs/cloudinary-photos-app/components/icons` | React Lib      | 16.1  | 16    | 0.01MB  | TS       | ❌ No               |
| 9    | `libs/cloudinary-photos-app/utils`            | TS Lib         | 11.0  | 11    | <0.01MB | TS       | ❌ No               |
| 10   | `libs/cloudinary-photos-app/global-store`     | Jotai Store    | 10.0  | 10    | <0.01MB | TS       | ❌ No               |
| 11   | `libs/cloudinary-photos-app/types`            | Type Defs      | 7.0   | 7     | <0.01MB | TS       | ❌ No               |

---

## Detailed Analysis

### ✅ HIGH PRIORITY: Create AGENTS.md

#### 1. `libs/wealth-management/` (Score: 357.6)

**Why**: 348 TypeScript files = **highest code complexity in monorepo**

| Metric        | Value                     |
| ------------- | ------------------------- |
| Files         | 348 (.ts: 225, .tsx: 119) |
| Size          | 0.96MB                    |
| Entry Point   | `src/index.ts`            |
| Targets       | `lint`, `test`            |
| Local Configs | `tsconfig.json` only      |

**Unique Patterns**:

- Likely domain-specific modules (AI, crypto, portfolio logic based on context)
- No build/export targets → pure library logic
- High file count with minimal size = dense, focused modules
- Extensive test suite inference (test target present)

**Recommended AGENTS.md Sections**:

```
## Architecture Overview
- Module organization (348 files across X directories)
- Domain layers (likely: AI, Crypto, Portfolio, Utilities)

## Entry Points & Exports
- Primary export: src/index.ts
- Public API scope & guarantees

## Type System & Validation
- Core types & interfaces
- Zod usage patterns (appears in root deps)

## Testing Strategy
- Jest configuration
- Mocking patterns for external APIs (AI SDKs, crypto feeds)

## Common Anti-patterns
- Circular dependency risks
- State mutation boundaries

## Integration Points
- How wealth-management app consumes this lib
- How vnstock-server might extend it (if crypto overlap)
```

---

#### 2. `apps/wealth-management/` (Score: 174.8)

**Why**: Complex Next.js app with **custom setup targets** (AI, OAuth, Auth) beyond standard patterns

| Metric        | Value                                                                                       |
| ------------- | ------------------------------------------------------------------------------------------- |
| Files         | 164 (.tsx: 91, .ts: 55, .json: 5)                                                           |
| Size          | 1.08MB                                                                                      |
| Entry Point   | Next.js app (no src/index)                                                                  |
| Targets       | `build`, `serve`, `serve-all`, `lint`, `test`, **`setup-ai`, `setup-oauth`, `verify-auth`** |
| Local Configs | `tsconfig.json`, `.eslintrc.json`, `next.config.ts`                                         |

**Unique Patterns**:

- Custom setup targets indicate integration complexity (OAuth flow, AI model initialization)
- Uses multiple AI SDKs (@ai-sdk/anthropic, @ai-sdk/azure, @ai-sdk/google, @ai-sdk/openai)
- Consumes wealth-management-lib for domain logic
- Environment-sensitive (setup tasks)

**Recommended AGENTS.md Sections**:

```
## Custom Setup Targets
- setup-ai: Initialize AI models & API keys
- setup-oauth: OAuth provider configuration
- verify-auth: Auth validation & token refresh logic

## OAuth & Authentication
- Auth flow (providers?)
- Session management
- Token storage & refresh

## AI Integration
- Which AI SDKs are active vs unused
- Prompt patterns & model selection
- Cost/rate limiting considerations

## Directory Structure
- Pages layout (app/pages structure for Next.js 16)
- Component hierarchy
- Shared layouts from wealth-management-lib

## Environment Variables & Secrets
- Required .env vars for OAuth & AI
- Development vs production setup

## State Management
- Jotai from cloudinary-photos-app or internal?
- Server state vs client state boundaries

## Common Pitfalls
- Hardcoded API keys
- OAuth redirect URL misconfigurations
- AI model context window management
```

---

#### 3. `apps/vnstock-server/` (Score: 127.6)

**Why**: **Only Python app in workspace** - isolated ecosystem requiring separate dev context

| Metric        | Value                                                                 |
| ------------- | --------------------------------------------------------------------- |
| Files         | 118 (.md: 80, .py: 32, .txt: 2)                                       |
| Size          | 0.96MB                                                                |
| Entry Point   | Flask/FastAPI app (inspect main.py or app.py)                         |
| Targets       | `install`, `serve`, `test`                                            |
| Local Configs | `requirements.txt` (no pyproject.toml, setup.py, or pytest.ini found) |

**Unique Patterns**:

- Python backend completely separate from TS/Node stack
- 80 markdown files suggest extensive docs → understand what they cover first
- Minimal infrastructure config (just requirements.txt)
- May use vnstock library for Vietnamese stock market data

**Recommended AGENTS.md Sections**:

```
## Python Environment Setup
- Virtual environment creation (.venv)
- Python version (infer from CI or defaults)
- Dependency installation & updates

## Project Structure
- Entry point (main.py, app.py, or __main__.py?)
- Module organization (api/, models/, services/)
- Database/ORM setup (if any)

## Dependency Management
- requirements.txt overview
- Pinned versions & compatibility notes
- Dev dependencies (pytest, linters)

## Server Patterns
- Framework (Flask, FastAPI, Django, or other?)
- Route organization
- Middleware/auth patterns

## Testing
- pytest configuration
- Mocking strategies
- Test data fixtures

## Integration with Monorepo
- How TypeScript apps call this server
- API contract (REST, gRPC, WebSocket?)
- Deployment context (Docker?)

## Common Pitfalls
- Import path confusion (PYTHONPATH)
- Async vs sync conflicts
- Virtual env not activated errors
```

---

### ⚠️ BORDERLINE: Consider Only If High Context-Switch

#### `apps/cloudinary-photos-app/` (Score: 66.2)

**Why Skipped**: Moderate complexity, well-structured, uses shared libs effectively

| Metric  | Value                                                |
| ------- | ---------------------------------------------------- |
| Files   | 56 (.tsx: 29, .ts: 10)                               |
| Size    | 1.02MB                                               |
| Targets | Standard: `build`, `serve`, `export`, `test`, `lint` |

**Decision**: ❌ Skip for now

- Clear Next.js patterns
- Delegates UI to shared libs (cloudinary-photos-app-components-\*)
- No custom setup or special targets
- Can be documented inline or in component lib docs

**If teams frequently context-switch to this app**, add later.

---

### ✅ SAFE TO SKIP: Trivial Complexity

#### `libs/cloudinary-photos-app/*` (Scores: 7–27)

**Why Skipped**: Self-documenting, minimal code, clear patterns

| Library            | Score | Reason                                      |
| ------------------ | ----- | ------------------------------------------- |
| `components/ui`    | 27.3  | Tested UI lib, straightforward              |
| `components/icons` | 16.1  | Icon library, obvious usage                 |
| `utils`            | 11.0  | Utility functions, code is self-explanatory |
| `global-store`     | 10.0  | Simple Jotai store, minimal state           |
| `types`            | 7.0   | Type definitions only                       |

#### `libs/portfolio-landpage/components` (Score: 30.2)

**Why Skipped**:

- Simple shared components (lint-only, no tests/build)
- Used by one app (portfolio-landpage)
- Code is self-documenting

---

## Directory Tree with Recommendations

```
monorepo/
├── .agents/
│   └── AGENTS.md  ← Root level, covers Nx setup & workspace structure
│
├── apps/
│   ├── cloudinary-photos-app/        (66.2) ❌ Skip
│   │   ├── tsconfig.json
│   │   ├── next.config.js
│   │   └── .eslintrc.json
│   │
│   ├── portfolio-landpage/           (588.9) ❌ Skip (asset-heavy)
│   │   ├── tsconfig.json
│   │   ├── jest.config.ts
│   │   ├── next.config.js
│   │   └── .eslintrc.json
│   │
│   ├── vnstock-server/               (127.6) ✅ ADD AGENTS.md
│   │   ├── requirements.txt
│   │   └── [80 markdown docs + 32 Python files]
│   │
│   └── wealth-management/            (174.8) ✅ ADD AGENTS.md
│       ├── tsconfig.json
│       ├── next.config.ts
│       ├── .eslintrc.json
│       └── [Custom targets: setup-ai, setup-oauth, verify-auth]
│
└── libs/
    ├── cloudinary-photos-app/
    │   ├── components/
    │   │   ├── icons/                (16.1) ❌ Skip
    │   │   ├── ui/                   (27.3) ❌ Skip
    │   │   └── layout/               (?) ❌ Skip
    │   ├── global-store/             (10.0) ❌ Skip
    │   ├── types/                    (7.0)  ❌ Skip
    │   └── utils/                    (11.0) ❌ Skip
    │
    ├── portfolio-landpage/
    │   └── components/               (30.2) ❌ Skip
    │
    └── wealth-management/            (357.6) ✅ ADD AGENTS.md
        └── [348 TypeScript files, highest complexity]
```

---

## Scoring Methodology

**Complexity Score = Files + (Size_MB × 10)**

Rationale:

- **Files**: Code density (more files = more modules to understand)
- **Size**: Filtered size (excludes node_modules, dist, .next, **pycache**)
- **Weight**: Assets (images, 3D models) inflate score unfairly → size weight < 1x

**Threshold for AGENTS.md**:

- Score > 100 AND
- Has unique patterns or cross-cutting concerns OR
- Language isolation (Python in TS monorepo) OR
- Custom targets beyond standard framework patterns

---

## Config Files Found

### All Projects

- `.eslintrc.json` (root + per-project overrides)
- `tsconfig.json` (root + per-project extensions)

### Apps

- **Next.js apps**: `next.config.js` or `next.config.ts`
- **Test runners**: `jest.config.ts` (portfolio-landpage, cloudinary-photos-app)

### Python

- `requirements.txt` only (no setuptools, pyproject.toml, or tox.ini)

### Root

- `jest.config.ts` (workspace preset)
- `jest.preset.js` (for Nx)
- `nx.json` (Nx configuration)
- `tsconfig.base.json` (monorepo base types)

---

## Entry Points by Type

| Type            | Pattern                                      | Example                                  |
| --------------- | -------------------------------------------- | ---------------------------------------- |
| **Next.js App** | No explicit src/index                        | App router in `app/` or `pages/`         |
| **TS Lib**      | `src/index.ts`                               | libs/cloudinary-photos-app/utils         |
| **React Lib**   | `src/index.ts` or `src/index.tsx`            | libs/cloudinary-photos-app/components/ui |
| **Python App**  | Likely `main.py`, `app.py`, or `__main__.py` | TBD in vnstock-server                    |

---

## Anti-patterns to Document (Post-Analysis)

When writing AGENTS.md for each selected directory:

### `libs/wealth-management/` Anti-patterns

1. Circular imports between domain modules
2. State mutation outside of defined update functions
3. API call side effects in selectors
4. Missing null/undefined guards on AI SDK responses

### `apps/wealth-management/` Anti-patterns

1. Hardcoding OAuth redirect URIs (should be env-based)
2. AI context window overflow (no length validation before sending)
3. Forgetting to `.env` OAuth secrets in CI/CD
4. Race conditions in auth token refresh

### `apps/vnstock-server/` Anti-patterns

1. Blocking I/O in sync routes (use async/await)
2. Database connection leaks on errors
3. PYTHONPATH not set when importing across packages
4. Requirements.txt without pinned versions (reproducibility)

---

## Next Steps

1. **Immediate** (This Week):
   - Create `apps/wealth-management/AGENTS.md`
   - Create `apps/vnstock-server/AGENTS.md`
   - Create `libs/wealth-management/AGENTS.md`

2. **Follow-up** (Optional):
   - Add inline comments to wealth-management & vnstock-server
   - Document expected environment variables
   - Create runbook for setup-ai & setup-oauth

3. **Validation**:
   - Verify vnstock-server entry point (inspect actual main file)
   - Review wealth-management AI SDK usage (which are actually used?)
   - Check if wealth-management-lib has separate domain modules

---

## Files Generated for Reference

- This analysis: `/tmp/MONOREPO_ANALYSIS.md`
- Raw scores: `/tmp/complexity_score.py` (Python)
- JSON output: Available on request
