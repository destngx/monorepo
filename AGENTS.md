## Engineering Conventions (Universal)

### Git Worktrees (Required for Every Feature)

Multiple agent sessions work in this repo concurrently. Never do feature work in the primary checkout (`~/projects/monorepo`) or on a branch another session already has checked out - uncommitted changes from different features will collide.

- **Start every new feature in its own worktree** with `gwf`, branched from an up-to-date `main`:
  ```sh
  git fetch origin
  cd "$(gwf create <type>/<app>-<topic> origin/main)"
  ```
  `gwf create` puts the worktree at `../monorepo-worktrees/<app>-<topic>`, runs `.gwf/setup` (installs `node_modules` from the shared pnpm store), and prints the worktree path. If the branch is already checked out somewhere, it prints that worktree's path instead; do not work there if another session owns it.
- **Without `gwf`** (not on PATH): `git worktree add -b <type>/<app>-<topic> ../monorepo-worktrees/<app>-<topic> origin/main`, `cd` into it, then run `.gwf/setup`.
- **Location**: always `../monorepo-worktrees/<app>-<topic>` (sibling of the repo, never nested inside it, so Nx and file watchers do not pick it up).
- **Check first**: run `gwf ls` (or `git worktree list`) and `git status` before editing. If the checkout has uncommitted changes unrelated to your task, stop and move to (or create) the correct worktree instead of editing alongside them.
- **One feature = one branch = one worktree.** Several agents may work on the same app at once, each in its own worktree. Only touch files belonging to your feature.
- **Cleanup** after merge: `git worktree remove ../monorepo-worktrees/<app>-<topic>` and `git branch -d <branch>`. `gwf rm` and `gwf gc` are interactive and meant for the user, not agents.

### Runtime and Tooling

- **Node**: Use `pnpm run` and `pnpx nx` for package management and task execution.
- **Workflow**: Default to **TDD + BDD**. Always verify changes with unit or integration tests.
- **Toolchain caches stay global**: never redirect `GOCACHE`, `GOMODCACHE`, `GOPATH`, the pnpm store, or other toolchain caches into the repo or a worktree. The global caches are shared across worktrees and safe for concurrent agents. If a cache write fails, stop and report it instead of working around it.

### Security and Env

- `.env.local` is for runtime input ONLY. Never commit, print, or log secrets.
- Use `./tmp` for temporary files; avoid root `/tmp`.

## Anti-Patterns

- **No NPM/Yarn/bun**: Use `pnpm` exclusively in this workspace.
- **No Error Swallowing**: Always handle or wrap errors properly.
- **Check Docs First**: Review the corresponding `docs/*/AGENTS.md` before implementing architectural changes.
