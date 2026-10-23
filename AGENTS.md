## Engineering Conventions (Universal)

### Git Worktrees (Required for Every Feature)

Multiple agent sessions work in this repo concurrently. Never do feature work in the primary checkout (`~/projects/monorepo`) or on a branch another session already has checked out - uncommitted changes from different features will collide.

- **Start every new feature in its own worktree** with `gwf`, branched from an up-to-date `main`:
  ```sh
  git fetch origin
  cd "$(gwf create <type>/<app>-<topic> origin/main)"
  ```
  `gwf create` puts the worktree at `../monorepo-worktrees/<app>-<topic>`, and prints the worktree path. If the branch is already checked out somewhere, it prints that worktree's path instead; do not work there if another session owns it.
- **Without `gwf`** (not on PATH): `git worktree add -b <type>/<app>-<topic> ../monorepo-worktrees/<app>-<topic> origin/main`, `cd` into it, then run `pnpm install --prefer-offline` (links `node_modules` from the shared pnpm store).
- **Location**: always `../monorepo-worktrees/<app>-<topic>` (sibling of the repo, never nested inside it, so Nx and file watchers do not pick it up).
- **Check first**: run `gwf ls` (without `gwf`: `git worktree list`) and `git status` before editing. If the checkout has uncommitted changes unrelated to your task, stop and move to (or create) the correct worktree instead of editing alongside them.
- **One feature = one branch = one worktree.** Several agents may work on the same app at once, each in its own worktree. Only touch files belonging to your feature.
- **Cleanup** after merge, from the primary checkout: `gwf rm -y -D <branch>` removes the worktree, deletes the branch with `git branch -d` and prunes. It skips a worktree with uncommitted changes or one in use by another process; report that instead of forcing it. Then run `git pull --ff-only` in the primary checkout, only when it is on `main` with a clean `git status`; otherwise skip it and say so. Interactive `gwf rm` (no `-y`) and `gwf gc` are for the user, not agents. Without `gwf`: `git worktree remove ../monorepo-worktrees/<app>-<topic>` and `git branch -d <branch>`.

### Runtime and Tooling

- **Node**: Use `pnpm run` and `pnpx nx` for package management and task execution.
- **Workspace binaries**: Nx targets and scripts call tools installed in the workspace with `pnpm exec <tool>` (eslint, vite, vitest, svelte-check), never `pnpx <tool>`: `pnpx` is `pnpm dlx`, which downloads the latest release and ignores the pinned version and its peers (it once ran ESLint 10 against a 9 config for 20+ minutes, and vitest without jsdom).
- **Workflow**: Default to **TDD + BDD**. Always verify changes with unit or integration tests.
- **Python**: uv. The apps are members of the uv workspace in the root `pyproject.toml` and share one root `uv.lock`, so a package two apps use has one version and one copy in the uv cache. Nx targets run `uv run --locked` with `UV_PROJECT_ENVIRONMENT=apps/<app>/.venv`, which creates or syncs that app's own `.venv` on every run, so there is no install step. To change dependencies, run `uv add --no-sync <name>` (`--dev` for test tools) in the app directory, or edit its `pyproject.toml` and run `uv lock`, then commit `uv.lock`. A new Python app adds itself to `members`. uv and Python come from `.mise.toml`.
- **Toolchain caches stay global**: never redirect `GOCACHE`, `GOMODCACHE`, `GOPATH`, the pnpm store, or other toolchain caches into the repo or a worktree. The global caches are shared across worktrees and safe for concurrent agents. If a cache write fails, stop and report it instead of working around it.

### Security and Env

- `.env.local` is for runtime input ONLY. Never commit, print, or log secrets.
- Use `./tmp` for temporary files; avoid root `/tmp`.

## Anti-Patterns

- **No NPM/Yarn/bun**: Use `pnpm` exclusively in this workspace.
- **No Error Swallowing**: Always handle or wrap errors properly.
- **Check Docs First**: Review the corresponding `docs/*/AGENTS.md` before implementing architectural changes.
- **UI work**: Read `docs/ui/README.md` before building or restyling any web UI. A new UI app follows its section 12.
