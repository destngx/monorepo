# Monorepo Knowledge Base

**Generated**: 2026-03-27  
**Commit**: 3d4e52e  
**Branch**: main

---

## OVERVIEW

Nx-based monorepo with **Next.js portfolio apps** (TypeScript/React) and **Python backend** (vnstock data analytics). Mixed-language workspace serving portfolio landing page, photo gallery, wealth management dashboard, and stock market data aggregation.

**Stack**: Nx 19+, Next.js 14, React 18, TypeScript, Python 3.12, Jest, TurboRepo caching

---

## WHERE TO LOOK

| Task                       | Location                                                    | Tech             | Status           |
| -------------------------- | ----------------------------------------------------------- | ---------------- | ---------------- |
| Wealth management features | `apps/wealth-management` + `libs/wealth-management`         | Next.js + React  | 🔄 Active        |
| Shared wealth lib (FSD)    | `libs/wealth-management/src`                                | TypeScript       | 📦 Stable        |
| Stock data integration     | `apps/vnstock-server`                                       | Python + vnstock | 📚 See AGENTS.md |
| Cloud photo gallery        | `apps/cloudinary-photos-app` + `libs/cloudinary-photos-app` | Next.js + React  | ✅ Stable        |
| Portfolio landing          | `apps/portfolio-landpage` + `libs/portfolio-landpage`       | Next.js + React  | ✅ Stable        |

---

## STRUCTURE

```
monorepo/
├── apps/                           # Standalone applications
│   ├── wealth-management/          # Main Next.js dashboard (AI chat, investing, goals)
│   ├── vnstock-server/             # Python backend (stock data crawler)
│   ├── cloudinary-photos-app/      # Gallery web app
│   └── portfolio-landpage/         # Portfolio showcase
├── libs/                           # Shared libraries (npm workspaces)
│   ├── wealth-management/          # Reusable wealth features (FSD architecture)
│   ├── cloudinary-photos-app/      # UI components + utils for gallery
│   └── portfolio-landpage/         # Portfolio components
├── docs/                           # Documentation
│   ├── tasks/                      # Task tracking
│   └── wealth-management/          # Wealth app architecture docs
├── tools/                          # Monorepo tooling
├── deployments/                    # Docker compose configs
├── nx.json                         # Nx configuration
├── tsconfig.base.json              # TypeScript base config
└── package.json                    # Root workspace
```

---

## CONVENTIONS

### TypeScript Workspace

**Entry Points**: Each app/lib has `package.json` or `tsconfig.json`  
**Module Format**: ESM (Next.js defaults to `type: "module"`)  
**Path Aliases**: Configured in `tsconfig.base.json`

```json
{
  "paths": {
    "@wealth-management/*": ["libs/wealth-management/src/*"],
    "@cloudinary-photos-app/*": ["libs/cloudinary-photos-app/*"]
  }
}
```

### Linting & Formatting

- **Linter**: ESLint (`.eslintrc.json` per package)
- **Formatter**: Prettier (`.prettierrc`)
- **Pre-commit**: Husky + lint-staged

### Testing

- **Jest** for unit/integration tests
- **Cypress** for e2e (wealth-management)
- Config: `jest.config.ts`, `jest.preset.js` (root)

### Python (vnstock-server)

- **Virtual Env**: `.venv/` in project root
- **Package Mgmt**: pip with `requirements.txt`
- **Tests**: pytest (`.pytest_cache/`)

---

## KEY PATTERNS

### Nx Build Graph

Commands execute with dependency tracking:

```bash
npm run build              # Build single app
npm run build:all         # Build all apps (respects dependencies)
npm run test:all          # Test all projects
nx run-many -t lint      # Lint all
```

**Important**: Nx caches build outputs. Clear with `nx reset`.

### Workspace Packages

Shared libs resolved via npm workspaces (no manual tsconfig paths):

```typescript
// ✅ OK: Import from @-scoped packages
import { useAccount } from '@wealth-management/features/accounts';

// ❌ WRONG: Direct relative imports between apps
import { X } from '../../../libs/wealth-management/src';
```

### Error Handling (Wealth App)

All errors inherit from `AppError` (never swallow errors):

```typescript
import { AppError } from "@wealth-management/utils/errors";

// ✅ Proper
throw new AppError("Invalid account", { code: "INVALID_ACCOUNT" });

// ❌ Wrong
catch (e) { /* silently fail */ }
```

---

## ANTI-PATTERNS (THIS PROJECT)

**NEVER:**

- Bypass vnstock_news internals (use library methods only)
- Import across feature boundaries directly (use public APIs)
- Swallow errors without logging
- Use relative imports for cross-workspace deps
- Commit `.env.local` or credential files
- Pass `site_name` strings directly (use structured configs)

**DO NOT:**

- Assume sites have RSS feeds without checking
- Patch around tsconfig paths (use `npm link`)
- Run `terraform init` with backend config (GitHub Actions handles it)

---

## UNIQUE STYLES

**Wealth Management**:

- **FSD (Feature-Sliced Design)**: `features/{feature}/ui`, `features/{feature}/model`
- **AI Integration**: System prompts, tool definitions, streaming responses
- **Google Sheets**: Real-time caching via Upstash Redis

**Vnstock Server**:

- **Agent-based**: Custom AGENTS.md rules per skill (news, migration, architecture)
- **Documentation-heavy**: Comprehensive docs for Vietnamese stock ecosystem

---

## COMMANDS

```bash
# Development
npm run serve                 # Serve wealth-management
npm run serve:all           # Serve all apps

# Build
npm run build               # Build single app (interactive)
npm run build:all          # Build all apps in dependency order
nx build wealth-management # Build specific app

# Testing
npm run test:all           # Run all tests
npm run e2e:all           # Run e2e tests
nx test {project}         # Test specific project

# Linting
npm run lint:all          # Lint all projects

# Git Commit
npm run cz                # Commitizen-driven commit workflow

# Release
npm run release           # Semantic versioning + changelog
```

---

## NOTES & GOTCHAS

**Wealth Management Refactoring**:

- Recent API extraction to `libs/wealth-management` (commit 3d4e52e)
- Old imports from `apps/` may still exist—use migration guide in `docs/`

**Python Dependencies**:

- vnstock libraries prefer `~/.venv` (shared across projects)
- Local `.venv` at `apps/vnstock-server/.venv/` for isolation

**Build Caching**:

- Nx caches aggressively; clear with `nx reset` if experiencing stale output
- GitHub Actions uses remote cache (S3)

**Type Safety**:

- No `as any` or `@ts-ignore` — resolve types properly
- Missing modules? Check workspace linking via `/link-workspace-packages` skill

---

## QUICK REFERENCE

| Workflow                  | Command                                      |
| ------------------------- | -------------------------------------------- |
| Add feature to wealth app | See `docs/wealth-management/ARCHITECTURE.md` |
| Integrate new vnstock API | See `apps/vnstock-server/AGENTS.md`          |
| Fix type errors           | `tsx` compiler, LSP, `tsc --noEmit`          |
| Debug Nx build failures   | `nx run-many -t build --verbose`             |
| Add workspace package     | Use `npm link` or workspace protocol         |
