---
name: rebase-merge
description: Land a feature branch on main locally with linear history - rebase the branch onto origin/main, fast-forward main to it (no merge commit), push, watch the deploy, then remove the worktree and branch. Use when the user asks to rebase-merge, "merge no ff", or land a branch/PR on main without the GitHub merge button.
argument-hint: '[branch] (defaults to the current branch)'
disable-model-invocation: true
---

# Rebase merge to main

Lands a branch on `main` the way GitHub's "Rebase and merge" would, but from the local checkout. `main` only
fast-forwards: the repo ruleset rejects merge commits ("Merge commits are not allowed in this repository"), so
`git merge --no-ff` is never used, even when the user says "no ff". Running this skill is the user's approval to
push to `main`, to force-push the feature branch with `--force-with-lease`, and to delete the branch (local and
remote) and its worktree once the merge and deploy are verified.

Branch: `$ARGUMENTS`, or the current branch when empty.

## Steps

1. **Locate the branch.** Run `git worktree list`. Work inside the branch's own worktree; never check out `main`
   or switch branches in the primary checkout (other sessions use it). Stop if `git status --short` shows
   uncommitted changes, and ask the user to commit them first.

2. **Rebase onto main.**

   ```sh
   git fetch origin
   git rebase origin/main
   ```

   On conflicts, stop and report them; do not resolve them silently. If the rebase rewrote commits, run the
   affected checks (tests, lint, build) again, then update the PR branch so GitHub can mark the PR merged:

   ```sh
   git push --force-with-lease origin <branch>
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
   git worktree remove <worktree>      # refuses if it has uncommitted changes; report, never --force
   git branch -d <branch>              # -d, not -D: refuses unless fully merged
   git push origin --delete <branch>
   git worktree prune
   ```

   Ignored files (`node_modules`, build output) go with the worktree; that is expected.

7. **Report.** The new `main` SHA, the PR state, the deploy URL and status, what the live check covered, and
   what was cleaned up.
