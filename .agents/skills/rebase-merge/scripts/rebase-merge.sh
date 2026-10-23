#!/usr/bin/env bash
# Mechanical steps of the rebase-merge skill. Each subcommand prints key=value lines on stdout, messages on stderr,
# and stops with a distinct exit code so the agent only has to make the judgement calls between steps.
#
#   prepare [branch]   rebase the branch's worktree onto origin/main and push the branch
#   land [branch]      fast-forward origin/main to the pushed branch and verify it
#   deploy [sha]       wait for the production deploy of every app with a vercel.json (default: origin/main)
#   cleanup <branch>   remove the merged branch's worktree, local and remote branch, then pull the primary checkout
#
# Exit codes:
#   0 done            1 usage or unexpected error   2 precondition not met   3 rebase conflict
#   4 main moved      5 push rejected                6 verification failed    7 deploy failed or missing
#   8 cleanup refused (worktree dirty or in use)
#
# Env: REBASE_MERGE_POLL_SECONDS (default 10), REBASE_MERGE_DEPLOY_TIMEOUT (seconds per app, default 600),
# REBASE_MERGE_PR_TRIES (default 6).
set -euo pipefail

POLL_SECONDS="${REBASE_MERGE_POLL_SECONDS:-10}"
DEPLOY_TIMEOUT="${REBASE_MERGE_DEPLOY_TIMEOUT:-600}"
PR_TRIES="${REBASE_MERGE_PR_TRIES:-6}"

kv() { printf '%s=%s\n' "$1" "$2"; }

die() {
  local code="$1"
  shift
  echo "rebase-merge: $*" >&2
  exit "$code"
}

primary_checkout() {
  git worktree list --porcelain | awk '/^worktree /{print substr($0, 10); exit}'
}

worktree_for() {
  git worktree list --porcelain |
    awk -v ref="branch refs/heads/$1" '/^worktree /{path = substr($0, 10)} $0 == ref {print path; exit}'
}

# Resolves the branch argument (default: the current branch) and its worktree into $branch and $wt.
resolve_branch_worktree() {
  branch="${1:-$(git branch --show-current)}"
  [[ -n "$branch" ]] || die 2 "no branch given and HEAD is detached"
  [[ "$branch" != main ]] || die 2 "refusing to land main onto itself; name the feature branch"
  wt="$(worktree_for "$branch")"
  [[ -n "$wt" ]] || die 2 "no worktree has $branch checked out; create one with gwf"
  [[ "$wt" != "$(primary_checkout)" ]] || die 2 "$branch is checked out in the primary checkout; move it to a worktree"
}

require_clean() {
  local dirty
  dirty="$(git -C "$1" status --porcelain)"
  [[ -z "$dirty" ]] || die 2 "uncommitted changes in $1, commit them first:"$'\n'"$dirty"
}

vercel_apps_at() {
  git ls-tree -r --name-only "$1" | awk '/(^|\/)vercel\.json$/ {sub(/\/?vercel\.json$/, ""); print}'
}

cmd_prepare() {
  resolve_branch_worktree "${1:-}"
  require_clean "$wt"
  git -C "$wt" fetch -q origin

  local before after rebase_out conflicts
  before="$(git -C "$wt" rev-parse HEAD)"
  if ! rebase_out="$(git -C "$wt" rebase origin/main 2>&1)"; then
    conflicts="$(git -C "$wt" diff --name-only --diff-filter=U)"
    git -C "$wt" rebase --abort
    if [[ -z "$conflicts" ]]; then
      die 1 "rebase failed without conflicts, aborted:"$'\n'"$rebase_out"
    fi
    while IFS= read -r f; do kv conflict "$f"; done <<<"$conflicts"
    die 3 "rebase onto origin/main conflicts; aborted, the branch is unchanged"
  fi
  after="$(git -C "$wt" rev-parse HEAD)"

  local commits
  commits="$(git -C "$wt" rev-list --count origin/main..HEAD)"
  [[ "$commits" -gt 0 ]] || die 2 "$branch has no commits on top of origin/main, nothing to land"

  git -C "$wt" push -q --force-with-lease -u origin "$branch" || die 5 "pushing $branch was rejected"
  [[ "$(git -C "$wt" rev-parse "$branch")" == "$(git -C "$wt" rev-parse "refs/remotes/origin/$branch")" ]] ||
    die 6 "origin/$branch does not match the local branch after the push"

  kv branch "$branch"
  kv worktree "$wt"
  kv head "$after"
  kv commits "$commits"
  kv rewritten "$([[ "$before" == "$after" ]] && echo no || echo yes)"
}

pr_state() {
  local out i
  command -v gh >/dev/null || { echo "unknown gh-not-on-path"; return; }
  for ((i = 1; i <= PR_TRIES; i++)); do
    if ! out="$(gh pr view "$1" --json state,url -q '.state + " " + .url' 2>&1)"; then
      if [[ "$out" == *"no pull requests found"* ]]; then
        echo none
      else
        echo "unknown $out"
      fi
      return
    fi
    [[ "$out" != MERGED* ]] || break
    ((i == PR_TRIES)) || sleep "$POLL_SECONDS"
  done
  echo "$out"
}

cmd_land() {
  resolve_branch_worktree "${1:-}"
  require_clean "$wt"
  git -C "$wt" fetch -q origin

  local tip remote_tip old_main push_out
  tip="$(git -C "$wt" rev-parse "$branch")"
  remote_tip="$(git -C "$wt" rev-parse -q --verify "refs/remotes/origin/$branch" || true)"
  [[ "$remote_tip" == "$tip" ]] || die 2 "origin/$branch is not the local tip; run prepare first"
  git -C "$wt" merge-base --is-ancestor origin/main "$tip" ||
    die 4 "origin/main moved since prepare; rerun prepare"

  old_main="$(git -C "$wt" rev-parse origin/main)"
  if ! push_out="$(git -C "$wt" push origin "$tip:refs/heads/main" 2>&1)"; then
    if [[ "$push_out" == *non-fast-forward* || "$push_out" == *"fetch first"* ]]; then
      die 4 "origin/main moved during the push; rerun prepare"
    fi
    die 5 "push to main rejected:"$'\n'"$push_out"
  fi

  git -C "$wt" fetch -q origin
  [[ "$(git -C "$wt" rev-parse origin/main)" == "$tip" ]] || die 6 "origin/main is not $tip after the push"

  local pr
  pr="$(pr_state "$branch")"
  kv old_main "$old_main"
  kv main "$tip"
  kv pr_state "${pr%% *}"
  [[ "$pr" != *" "* ]] || kv pr_detail "${pr#* }"
  kv vercel_apps "$(vercel_apps_at "$tip" | paste -sd ' ' -)"
}

# Polls the production deployment of one app for a commit until it reaches a final state. Prints the state and URL.
deploy_state() {
  local project="$1" sha="$2" deadline json url state err
  deadline=$((SECONDS + DEPLOY_TIMEOUT))
  err="$(mktemp)"
  while :; do
    if json="$(vercel ls "$project" --prod -m "githubCommitSha=$sha" --format json 2>"$err")"; then
      url="$(jq -r '.deployments[0].url // empty' <<<"$json")"
      state="$(jq -r '.deployments[0].state // empty' <<<"$json")"
      case "$state" in
        READY | ERROR | CANCELED)
          rm -f "$err"
          echo "$state $url"
          return
          ;;
      esac
    fi
    if ((SECONDS >= deadline)); then
      [[ ! -s "$err" ]] || cat "$err" >&2
      rm -f "$err"
      echo "${state:+TIMEOUT_}${state:-NOT_FOUND} ${url:-}"
      return
    fi
    sleep "$POLL_SECONDS"
  done
}

cmd_deploy() {
  git fetch -q origin
  local sha apps app result state url failed=0
  sha="$(git rev-parse "${1:-origin/main}")"
  apps="$(vercel_apps_at "$sha")"
  kv sha "$sha"
  kv vercel_apps "$(paste -sd ' ' - <<<"$apps")"
  [[ -n "$apps" ]] || return 0
  command -v vercel >/dev/null || die 2 "vercel is not on PATH; ask the user instead of installing it"

  while IFS= read -r app; do
    result="$(deploy_state "$(basename "$app")" "$sha")"
    state="${result%% *}"
    url="${result#* }"
    case "$state" in
      READY) kv "deploy.$app" "READY https://$url" ;;
      CANCELED) kv "deploy.$app" "SKIPPED (ignored build step) https://$url" ;;
      *)
        kv "deploy.$app" "$state${url:+ https://$url}"
        [[ "$state" != ERROR ]] || vercel inspect --logs "$url" >&2 || echo "rebase-merge: vercel inspect --logs failed" >&2
        failed=1
        ;;
    esac
  done <<<"$apps"
  ((failed == 0)) || die 7 "a production deploy failed or did not finish in ${DEPLOY_TIMEOUT}s"
}

cmd_cleanup() {
  branch="${1:-}"
  [[ -n "$branch" ]] || die 1 "usage: cleanup <branch>"
  [[ "$branch" != main ]] || die 2 "refusing to delete main"
  local primary tip wt rm_out pnpm_out
  primary="$(primary_checkout)"
  cd "$primary"
  git fetch -q origin

  tip="$(git rev-parse -q --verify "refs/heads/$branch")" || die 2 "no local branch $branch"
  git merge-base --is-ancestor "$tip" origin/main || die 6 "$branch is not on origin/main; land it first"

  wt="$(worktree_for "$branch")"
  if [[ -n "$wt" ]]; then
    [[ "$wt" != "$primary" ]] || die 2 "$branch is checked out in the primary checkout"
    if command -v pnpm >/dev/null && ! pnpm_out="$( (cd "$wt" && pnpm exec nx daemon --stop) 2>&1)"; then
      echo "rebase-merge: stopping the Nx daemon in $wt failed, continuing:"$'\n'"$pnpm_out" >&2
    fi
    if command -v gwf >/dev/null; then
      rm_out="$(gwf rm -y -D "$branch" 2>&1)" || die 8 "gwf rm failed:"$'\n'"$rm_out"
    else
      rm_out="$(git worktree remove "$wt" 2>&1 && git branch -d "$branch" 2>&1 && git worktree prune 2>&1)" ||
        die 8 "removing the worktree failed:"$'\n'"$rm_out"
    fi
    [[ ! -d "$wt" ]] || die 8 "the worktree $wt is still there:"$'\n'"$rm_out"
    kv worktree_removed "$wt"
  fi
  if git show-ref -q --verify "refs/heads/$branch"; then
    git branch -d "$branch" >/dev/null || die 8 "git branch -d $branch refused"
  fi
  kv branch_deleted "$branch"

  if git ls-remote -q --exit-code --heads origin "$branch" >/dev/null; then
    git push -q origin --delete "$branch" || die 5 "deleting origin/$branch was rejected"
  fi
  kv remote_branch_deleted "$branch"

  if [[ "$(git branch --show-current)" != main ]]; then
    kv pulled "skipped (primary checkout is on $(git branch --show-current || echo 'a detached HEAD'))"
  elif [[ -n "$(git status --porcelain)" ]]; then
    kv pulled "skipped (primary checkout has uncommitted changes)"
  else
    git pull -q --ff-only || die 6 "git pull --ff-only failed in the primary checkout"
    kv pulled yes
  fi
  kv main "$(git rev-parse origin/main)"
}

case "${1:-}" in
  prepare | land | deploy | cleanup)
    cmd="$1"
    shift
    "cmd_$cmd" "$@"
    ;;
  *) die 1 "usage: rebase-merge.sh prepare|land|deploy|cleanup [branch|sha]" ;;
esac
