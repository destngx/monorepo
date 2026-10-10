## Engineering Conventions (Universal)

### Git Worktrees (Required for Every Feature)

Multiple agent sessions work in this repo concurrently. Never do feature work in the primary checkout (`~/projects/monorepo`) or on a branch another session already has checked out - uncommitted changes from different features will collide.

- **Start every new feature in its own worktree**, branched from an up-to-date `main`:
  ```sh
  git fetch origin
  git worktree add -b <type>/<app>-<topic> ../monorepo-worktrees/<app>-<topic> origin/main
  cd ../monorepo-worktrees/<app>-<topic>
  pnpm install   # node_modules / .venv are per-worktree
  ```
- **Location**: always `../monorepo-worktrees/<app>-<topic>` (sibling of the repo, never nested inside it, so Nx and file watchers do not pick it up).
- **Check first**: run `git worktree list` and `git status` before editing. If the checkout has uncommitted changes unrelated to your task, stop and move to (or create) the correct worktree instead of editing alongside them.
- **One feature = one branch = one worktree.** Only touch files belonging to your feature.
- **Cleanup** after merge: `git worktree remove ../monorepo-worktrees/<app>-<topic>` and `git branch -d <branch>`.

### Runtime and Tooling

- **Node**: Use `pnpm run` and `pnpx nx` for package management and task execution.
- **Workflow**: Default to **TDD + BDD**. Always verify changes with unit or integration tests.

### Security and Env

- `.env.local` is for runtime input ONLY. Never commit, print, or log secrets.
- Use `./tmp` for temporary files; avoid root `/tmp`.

## Anti-Patterns

- **No NPM/Yarn/bun**: Use `pnpm` exclusively in this workspace.
- **No Error Swallowing**: Always handle or wrap errors properly.
- **Check Docs First**: Review the corresponding `docs/*/AGENTS.md` before implementing architectural changes.
