---
name: rebase-merge
description: Land a feature branch on main locally with linear history - rebase the branch onto origin/main, fast-forward main to it (no merge commit), push, watch the deploy, then remove the worktree and branch. Use when the user asks to rebase-merge, "merge rebase", "proceed merge", "merge no ff", or land a branch/PR on main without the GitHub merge button.
argument-hint: '[branch] (defaults to the current branch)'
---

# Rebase merge to main

Lands a branch on `main` the way GitHub's "Rebase and merge" would, but from the local checkout. `main` only
fast-forwards: the repo ruleset rejects merge commits ("Merge commits are not allowed in this repository"), so
`git merge --no-ff` is never used, even when the user says "no ff". The user asking for a rebase merge (by
slash command or in plain words such as "merge rebase") is their approval to
push to `main`, to force-push the feature branch with `--force-with-lease`, and to delete the branch (local and
remote) and its worktree once the merge and deploy are verified.

Branch: `$ARGUMENTS`, or the current branch when empty.

## Steps

1. **Locate the branch.** Run `gwf ls` (without `gwf`: `git worktree list`). Work inside the branch's own
   worktree; never check out `main` or switch branches in the primary checkout (other sessions use it). Stop if
   `git status --short` shows uncommitted changes, and ask the user to commit them first.

2. **Rebase onto main.**

   ```sh
   git fetch origin
   git rebase origin/main
   ```

   On conflicts, stop and report them; do not resolve them silently. If the rebase rewrote commits, run the
   affected checks (tests, lint, build) again. Then push the branch, even when it has no PR: GitHub marks a PR
   merged from it, and the upstream lets `git branch -d` in cleanup check the branch against its pushed copy
   instead of the primary checkout's `main`, which may lag behind origin:

   ```sh
   git push --force-with-lease -u origin <branch>
   ```

   Confirm `git rev-parse <branch>` equals `git rev-parse origin/<branch>`.

3. **Fast-forward main and push.** From the same worktree, on a detached HEAD:

   ```sh
   git checkout --detach origin/main
   git merge --ff-only <branch>
   git push origin HEAD:main
   git checkout <branch>
   ```

   If `--ff-only` fails, `origin/main` moved after the rebase: go back to step 2. If the push is rejected by a
   ruleset (for example required reviews or status checks), report the exact message and stop; do not try to
   bypass it.

4. **Verify.**
   - `git fetch origin && git log --oneline -1 origin/main` shows the branch tip.
   - If the branch has a PR, `gh pr view <branch> --json state -q .state` is `MERGED` (GitHub can take a few
     seconds).

5. **Watch the deploy.** Find the apps the branch touched (`git diff --name-only <old-main>..origin/main`) and
   keep those with a `vercel.json`. For each, the push to `main` triggers a production deploy (unless its ignored
   build step decides the app is unaffected). Use the Vercel CLI read-only (never `vercel link`, which writes
   `.vercel/`). If `vercel` is not on PATH, ask the user rather than installing it:

   ```sh
   vercel ls <project> --prod        # find the deployment for the new main SHA
   vercel inspect --wait <url>       # block until it is Ready or Error
   vercel inspect --logs <url>       # on Error, or to check for new warnings
   ```

   When it is Ready, check the live site: the changed routes return 200 and show the change. If the deploy fails
   or the site is wrong, stop here, report it, and skip cleanup so the worktree stays available for a fix. With
   no Vercel app touched, go straight to cleanup.

6. **Clean up.** Only after the merge is verified and the deploy (if any) is good. Run from the primary checkout,
   never from inside the worktree being removed:

   ```sh
   cd <primary checkout>
   gwf rm -y -D <branch>               # removes the worktree, deletes the branch with -d, prunes
   git push origin --delete <branch>
   git pull --ff-only                  # only when on main with a clean `git status --short`
   ```

   `gwf rm -y` skips a worktree with uncommitted changes or one another process is using, and `-D` deletes the
   branch with `git branch -d`, which refuses unless it is fully merged. If either is refused, report it; never
   force-remove the worktree or use `git branch -D`. Without `gwf` on PATH, run `git worktree remove <worktree>`,
   `git branch -d <branch>` and `git worktree prune` instead. Ignored files (`node_modules`, build output) go
   with the worktree; that is expected.

   Then bring the primary checkout's `main` up to date. Pull only when it is on `main` and `git status --short`
   is empty; otherwise another session is working there, so skip the pull and report it. If `--ff-only` fails,
   report it; never merge or rebase the primary checkout.

7. **Report.** The new `main` SHA, the PR state, the deploy URL and status, what the live check covered, what was
   cleaned up, and whether the primary checkout was pulled.
