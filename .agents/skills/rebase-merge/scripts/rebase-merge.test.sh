#!/usr/bin/env bash
# Tests for rebase-merge.sh. Each case builds a bare origin, a primary checkout on main and a feature worktree in a
# temporary directory, with stub gh, vercel, gwf and pnpm binaries on PATH, so nothing touches the real remote.
# Run: bash .agents/skills/rebase-merge/scripts/rebase-merge.test.sh
set -uo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/rebase-merge.sh"
REPO_TMP="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/tmp"
mkdir -p "$REPO_TMP"
ROOT="$(mktemp -d "$REPO_TMP/rebase-merge-test.XXXXXX")"
trap 'rm -rf "$ROOT"' EXIT

passed=0
failed=0

fail() {
  echo "  FAIL: $*"
  failed=$((failed + 1))
  return 1
}

assert_eq() { [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains() { [[ "$1" == *"$2"* ]] || fail "$3: expected output to contain '$2', got: $1"; }

# Sets up $T/origin.git, $T/primary (main) and $T/wt (branch feat/x with one commit) and stub binaries in $T/bin.
setup() {
  T="$ROOT/$1"
  mkdir -p "$T/bin"
  git init -q --bare -b main "$T/origin.git"
  git clone -q "$T/origin.git" "$T/primary" 2>/dev/null
  git -C "$T/primary" config user.email test@example.com
  git -C "$T/primary" config user.name test
  git -C "$T/primary" config core.hooksPath /dev/null
  echo base >"$T/primary/base.txt"
  mkdir -p "$T/primary/apps/site"
  echo '{}' >"$T/primary/apps/site/vercel.json"
  git -C "$T/primary" add -A
  git -C "$T/primary" commit -qm "chore: base"
  git -C "$T/primary" push -q origin main
  git -C "$T/primary" worktree add -q -b feat/x "$T/wt" origin/main 2>/dev/null
  echo feature >"$T/wt/feature.txt"
  git -C "$T/wt" add -A
  git -C "$T/wt" commit -qm "feat: x"

  # gh: no PR unless $T/pr-state exists.
  cat >"$T/bin/gh" <<EOF
#!/usr/bin/env bash
[[ -f "$T/pr-state" ]] || { echo 'no pull requests found' >&2; exit 1; }
echo "\$(cat "$T/pr-state") https://example.test/pr/1"
EOF
  # vercel: answers ls from $T/vercel-state (missing file means no deployment yet), logs a line per call.
  cat >"$T/bin/vercel" <<EOF
#!/usr/bin/env bash
echo "\$*" >>"$T/vercel-calls"
case "\$1" in
  ls)
    if [[ -f "$T/vercel-state" ]]; then
      printf '{"deployments":[{"url":"site-abc.vercel.app","state":"%s"}]}\n' "\$(cat "$T/vercel-state")"
    else
      echo '{"deployments":[]}'
    fi
    ;;
  inspect)
    [[ "\$*" == *--logs* ]] && echo "build log line"
    exit 0
    ;;
esac
EOF
  # gwf: only "rm -y -D <branch>", implemented with plain git; refuses when $T/in-use exists.
  cat >"$T/bin/gwf" <<EOF
#!/usr/bin/env bash
[[ "\$1 \$2 \$3" == "rm -y -D" ]] || { echo "unexpected gwf \$*" >&2; exit 1; }
if [[ -f "$T/in-use" ]]; then echo "skip $T/wt: in use by: node (pid 4242)"; exit 0; fi
git worktree remove "$T/wt" && git branch -d "\$4" && git worktree prune
EOF
  cat >"$T/bin/pnpm" <<EOF
#!/usr/bin/env bash
echo "\$*" >>"$T/pnpm-calls"
EOF
  chmod +x "$T/bin/"*
}

# Runs the script with the stubs first on PATH. Sets $out and $code.
run() {
  local dir="$1"
  shift
  out="$(cd "$dir" && PATH="$T/bin:$PATH" REBASE_MERGE_POLL_SECONDS=1 REBASE_MERGE_DEPLOY_TIMEOUT=2 \
    bash "$SCRIPT" "$@" 2>&1)"
  code=$?
}

# Adds a commit on origin/main from a scratch clone, so the branch falls behind.
advance_main() {
  local file="$1" content="$2"
  git clone -q "$T/origin.git" "$T/scratch-$file" 2>/dev/null
  git -C "$T/scratch-$file" -c user.email=t@e -c user.name=t -c core.hooksPath=/dev/null \
    commit -q --allow-empty -m "chore: bump" 2>/dev/null
  echo "$content" >"$T/scratch-$file/$file"
  git -C "$T/scratch-$file" add -A
  git -C "$T/scratch-$file" -c user.email=t@e -c user.name=t -c core.hooksPath=/dev/null commit -qm "chore: $file"
  git -C "$T/scratch-$file" push -q origin main
}

test_prepare_pushes_branch_without_rewrite() {
  setup prepare-plain
  run "$T/wt" prepare
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "rewritten=no" "rewrite flag" || return
  assert_contains "$out" "commits=1" "commit count" || return
  assert_eq "$(git -C "$T/wt" rev-parse feat/x)" "$(git -C "$T/origin.git" rev-parse feat/x)" "pushed tip"
}

test_prepare_reports_rewrite_when_main_moved() {
  setup prepare-rewrite
  advance_main other.txt other
  run "$T/wt" prepare
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "rewritten=yes" "rewrite flag" || return
  git -C "$T/wt" merge-base --is-ancestor origin/main feat/x || fail "branch not on top of main"
}

test_prepare_refuses_dirty_worktree() {
  setup prepare-dirty
  echo wip >"$T/wt/wip.txt"
  run "$T/wt" prepare
  assert_eq "$code" 2 "exit code" || return
  assert_contains "$out" "wip.txt" "lists the dirty file"
}

test_prepare_aborts_on_conflict() {
  setup prepare-conflict
  advance_main feature.txt "main version"
  run "$T/wt" prepare
  assert_eq "$code" 3 "exit code" || return
  assert_contains "$out" "conflict=feature.txt" "lists the conflicting file" || return
  [[ ! -d "$(git -C "$T/wt" rev-parse --git-path rebase-merge)" ]] || fail "rebase left in progress"
}

test_prepare_refuses_branch_without_commits() {
  setup prepare-empty
  git -C "$T/wt" reset -q --hard origin/main
  run "$T/wt" prepare
  assert_eq "$code" 2 "exit code" || return
  assert_contains "$out" "nothing to land" "message"
}

test_prepare_finds_worktree_by_branch_name() {
  setup prepare-by-name
  run "$T/primary" prepare feat/x
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "worktree=$T/wt" "worktree path"
}

test_prepare_refuses_main() {
  setup prepare-main
  run "$T/primary" prepare main
  assert_eq "$code" 2 "exit code"
}

test_land_fast_forwards_main() {
  setup land-ok
  echo MERGED >"$T/pr-state"
  run "$T/wt" prepare
  run "$T/wt" land
  assert_eq "$code" 0 "exit code" || return
  assert_eq "$(git -C "$T/origin.git" rev-parse main)" "$(git -C "$T/wt" rev-parse feat/x)" "origin main tip" || return
  assert_contains "$out" "pr_state=MERGED" "PR state" || return
  assert_contains "$out" "vercel_apps=apps/site" "vercel apps" || return
  assert_eq "$(git -C "$T/wt" branch --show-current)" "feat/x" "worktree still on the branch"
}

test_land_without_pr() {
  setup land-no-pr
  run "$T/wt" prepare
  run "$T/wt" land
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "pr_state=none" "PR state"
}

test_land_reports_main_moved() {
  setup land-moved
  run "$T/wt" prepare
  advance_main other.txt other
  run "$T/wt" land
  assert_eq "$code" 4 "exit code" || return
  assert_contains "$out" "rerun prepare" "message"
}

test_land_refuses_unpushed_branch() {
  setup land-unpushed
  run "$T/wt" land
  assert_eq "$code" 2 "exit code" || return
  assert_contains "$out" "run prepare" "message"
}

test_deploy_ready() {
  setup deploy-ready
  echo READY >"$T/vercel-state"
  run "$T/wt" deploy
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "deploy.apps/site=READY https://site-abc.vercel.app" "deploy line" || return
  assert_contains "$(cat "$T/vercel-calls")" "ls site --prod -m githubCommitSha=" "looks up by SHA"
}

test_deploy_canceled_counts_as_skipped() {
  setup deploy-canceled
  echo CANCELED >"$T/vercel-state"
  run "$T/wt" deploy
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "deploy.apps/site=SKIPPED" "deploy line"
}

test_deploy_error_prints_logs() {
  setup deploy-error
  echo ERROR >"$T/vercel-state"
  run "$T/wt" deploy
  assert_eq "$code" 7 "exit code" || return
  assert_contains "$out" "deploy.apps/site=ERROR" "deploy line" || return
  assert_contains "$out" "build log line" "logs"
}

test_deploy_times_out_without_deployment() {
  setup deploy-missing
  run "$T/wt" deploy
  assert_eq "$code" 7 "exit code" || return
  assert_contains "$out" "deploy.apps/site=NOT_FOUND" "deploy line"
}

test_cleanup_removes_worktree_and_branches() {
  setup cleanup-ok
  run "$T/wt" prepare
  run "$T/wt" land
  run "$T/primary" cleanup feat/x
  assert_eq "$code" 0 "exit code" || return
  [[ ! -d "$T/wt" ]] || fail "worktree still exists" || return
  ! git -C "$T/primary" show-ref -q --verify refs/heads/feat/x || fail "local branch still exists" || return
  ! git -C "$T/origin.git" show-ref -q --verify refs/heads/feat/x || fail "remote branch still exists" || return
  assert_contains "$(cat "$T/pnpm-calls")" "exec nx daemon --stop" "stops the Nx daemon" || return
  assert_contains "$out" "pulled=yes" "pull" || return
  assert_eq "$(git -C "$T/primary" rev-parse main)" "$(git -C "$T/origin.git" rev-parse main)" "primary main updated"
}

test_cleanup_skips_pull_when_primary_dirty() {
  setup cleanup-dirty-primary
  run "$T/wt" prepare
  run "$T/wt" land
  echo other-session >"$T/primary/wip.txt"
  run "$T/primary" cleanup feat/x
  assert_eq "$code" 0 "exit code" || return
  assert_contains "$out" "pulled=skipped" "pull skipped"
}

test_cleanup_refuses_unmerged_branch() {
  setup cleanup-unmerged
  run "$T/wt" prepare
  run "$T/primary" cleanup feat/x
  assert_eq "$code" 6 "exit code" || return
  [[ -d "$T/wt" ]] || fail "worktree removed"
}

test_cleanup_reports_worktree_in_use() {
  setup cleanup-in-use
  run "$T/wt" prepare
  run "$T/wt" land
  touch "$T/in-use"
  run "$T/primary" cleanup feat/x
  assert_eq "$code" 8 "exit code" || return
  assert_contains "$out" "in use by: node (pid 4242)" "gwf message" || return
  git -C "$T/origin.git" show-ref -q --verify refs/heads/feat/x || fail "remote branch deleted before the worktree"
}

for t in $(declare -F | awk '{print $3}' | grep '^test_'); do
  echo "$t"
  if "$t"; then passed=$((passed + 1)); fi
done
echo "passed=$passed failed=$failed"
[[ $failed -eq 0 ]]
