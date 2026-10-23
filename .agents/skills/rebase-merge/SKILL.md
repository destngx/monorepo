---
name: rebase-merge
description: Land a feature branch on main locally with linear history - rebase the branch onto origin/main, fast-forward main to it (no merge commit), push, watch the deploy, then remove the worktree and branch. Use when the user asks to rebase-merge, "merge rebase", "proceed merge", "merge no ff", or land a branch/PR on main without the GitHub merge button.
argument-hint: '[branch] (defaults to the current branch)'
---

# Rebase merge to main

Lands a branch on `main` the way GitHub's "Rebase and merge" would, but from the local checkout. `main` only
fast-forwards: the repo ruleset rejects merge commits ("Merge commits are not allowed in this repository"), so
`git merge --no-ff` is never used, even when the user says "no ff". The user asking for a rebase merge (by
slash command or in plain words such as "merge rebase") is their approval to push to `main`, to force-push the
feature branch with `--force-with-lease`, and to delete the branch (local and remote) and its worktree once the
merge and deploy are verified.

Branch: `$ARGUMENTS`, or the current branch when empty.

`scripts/rebase-merge.sh` (repo path `.agents/skills/rebase-merge/scripts/rebase-merge.sh`) runs every git, gh and
Vercel command. Do not run those steps by hand. Each subcommand prints `key=value` lines and exits with a code that
says what to do next. Your part is the decisions between subcommands: which checks to rerun, whether the live site
is right, and which leftover processes to stop. The header of the script lists every exit code.

Run it exactly as written below, from a checkout's root, with no `cd` or env prefix in the same command. The
project settings allow only `bash .agents/skills/rebase-merge/scripts/rebase-merge.sh *`, and that rule is the
permission to push to `main`. The repo's `main` ruleset rejects force pushes and deleting `main` on the server. Its tests are in
`scripts/rebase-merge.test.sh`. Run them with `bash` after changing the script.

## Steps

1. **Prepare.** From the branch's worktree (or from anywhere, naming the branch):

   ```sh
   bash .agents/skills/rebase-merge/scripts/rebase-merge.sh prepare [branch]
   ```

   It refuses a dirty worktree, a branch with no commits and a branch in the primary checkout (exit 2). It
   rebases onto `origin/main` and pushes the branch with `--force-with-lease -u`. The push lets GitHub mark the
   PR merged, and lets `git branch -d` in cleanup check against the pushed copy.
   - Exit 2 means a precondition failed. Report it. For uncommitted changes, ask the user to commit first.
   - Exit 3 means the rebase conflicted. The script aborted it, so the branch is unchanged. Report the
     `conflict=` files and stop. Do not resolve conflicts silently.
   - `rewritten=yes` means `main` moved and the commits changed. Decide which checks the new base affects (the
     tests, lint and build of the projects the branch touches, plus any project that `main`'s new commits change
     under it). Run them in the worktree before landing. Fix or report failures, and do not land a red branch.

2. **Land.**

   ```sh
   bash .agents/skills/rebase-merge/scripts/rebase-merge.sh land [branch]
   ```

   It pushes the branch tip to `main`. The push is a fast-forward or nothing, because there is no force. Then it
   checks that `origin/main` equals the tip and waits for the PR to show `MERGED`.
   - Exit 4 means `origin/main` moved after prepare. Go back to step 1.
   - Exit 5 means a ruleset rejected the push (for example, required reviews or status checks). Report the exact
     message and stop. Do not try to bypass it.
   - `pr_state=OPEN` after the retries usually means GitHub is slow. Check again with
     `gh pr view <branch> --json state` before reporting it.

   Keep `old_main` and `vercel_apps` from the output for the next steps.

3. **Watch the deploy.** If `vercel_apps` is empty, go to step 4.

   ```sh
   bash .agents/skills/rebase-merge/scripts/rebase-merge.sh deploy
   ```

   For each app with a `vercel.json`, it waits for the production deployment of the new `main` SHA. It uses the
   Vercel CLI read-only (it never runs `vercel link`). `SKIPPED` means the app's ignored build step found the app
   unaffected, which is fine. Exit 7 means a deploy failed (the build log is on stderr) or never finished. Exit 2
   means `vercel` is not on PATH. In that case, ask the user instead of installing it.

   For each `READY` app, check the live site yourself. Find the routes the change affects
   (`git diff --name-only <old_main>..origin/main`). Open them on the production domain, and confirm that they
   return 200 and show the change. If the deploy failed or the site is wrong, stop, report it, and skip cleanup,
   so the worktree stays available for a fix.

4. **Clean up.** Only after the merge is verified and the deploy (if any) is good. First `cd` to the primary
   checkout, so your shell is not inside the worktree being removed:

   ```sh
   bash .agents/skills/rebase-merge/scripts/rebase-merge.sh cleanup <branch>
   ```

   It refuses a branch that is not on `origin/main` (exit 6). It stops the worktree's Nx daemon, runs
   `gwf rm -y -D <branch>` (or `git worktree remove` and `git branch -d` without `gwf`), and deletes the remote
   branch. It pulls the primary checkout only when it is on `main` and clean. Otherwise it prints
   `pulled=skipped`, because another session is working there. Ignored files (`node_modules`, build output) go
   with the worktree, which is expected.
   - Exit 8 means `gwf rm` skipped the worktree. If the message says `in use by: <name> (pid …)`, run
     `ps -o pid,command -p <pid>` to see what the process is. Stop it only if your own runs started it from this
     worktree's path (a dev server, a test engine). Then rerun cleanup. Never stop the user's processes or
     anything running from the primary checkout. If the worktree has uncommitted changes, or a process is not
     yours, report it. Never force-remove the worktree or use `git branch -D`.
   - Exit 6 from the pull means `--ff-only` failed. Report it. Never merge or rebase the primary checkout.

5. **Report.** The new `main` SHA, the PR state, the deploy URL and status, what the live check covered, what was
   cleaned up, and whether the primary checkout was pulled.
