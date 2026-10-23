#!/bin/zsh
# Tests for dev-cpu-watch. Sources the script and replaces its system calls
# (ps, lsof, the clock, notifications, git) with fakes.
setopt err_exit no_unset pipe_fail

TEST_DIR=${0:A:h}
source $TEST_DIR/../src/dev-cpu-watch

mkdir -p ${TEST_DIR:h:h:h}/tmp
WORK=$(mktemp -d "${TEST_DIR:h:h:h}/tmp/dev-cpu-watch-test.XXXXXX")
trap 'rm -rf $WORK' EXIT

passes=0
fails=0

assert_eq() {
  local actual=$1 expected=$2 name=$3
  if [[ $actual == "$expected" ]]; then
    passes=$((passes + 1))
  else
    fails=$((fails + 1))
    print -u2 -r -- "FAIL: $name"
    print -u2 -r -- "  expected: $expected"
    print -u2 -r -- "  actual:   $actual"
  fi
}

detect() {
  awk -v now=$1 -v threshold=25 -v runs=2 -v cooldown=3600 -f $APP_DIR/src/detect.awk $2 $3
}

# --- detect.awk ---------------------------------------------------------------

print -r -- "  101   0:01.50 /usr/bin/foo --flag" > $WORK/ps1
print -r -- "  102 123:45.67 /r/monorepo/apps/a/.venv/bin/python3 -m x" >> $WORK/ps1
print -r -- "  103 1-02:03:04 /bin/long" >> $WORK/ps1
print -r -- "  104   0:00.00" >> $WORK/ps1

out=$(detect 1000 /dev/null $WORK/ps1)
assert_eq "$(print -r -- $out | cut -f1,2 | tr '\t\n' ' |')" \
  "101 1.50|102 7425.67|103 93784.00|" \
  "parses MM:SS, MMM:SS and D-HH:MM:SS cpu times and skips lines without a command"
assert_eq "$(print -r -- $out | cut -f3-6 | sort -u)" $'0\t0\t0\t0' \
  "first run has no hot, pct or due values"
assert_eq "$(print -r -- $out | awk -F'\t' '$1==102 {print $7}')" \
  "/r/monorepo/apps/a/.venv/bin/python3 -m x" "keeps the full command"

state=$WORK/state1
printf 'at\t1000\n102\t100.00\t1\t0\t40\t/bin/py\n103\t50.00\t1\t0\t40\t/bin/idle\n104\t0.00\t1\t0\t40\t/bin/old\n' > $state
printf '  102   1:55.00 /bin/py\n  103   0:53.00 /bin/idle\n  104   1:00.00 /bin/new\n' > $WORK/ps2
out=$(detect 1300 $state $WORK/ps2)
assert_eq "$(print -r -- $out | awk -F'\t' '$1==102 {print $3,$5,$6}')" "0 5 0" \
  "5% over the interval is below the threshold and resets the count"
printf '  102   6:15.00 /bin/py\n' > $WORK/ps3
out=$(detect 1300 $state $WORK/ps3)
assert_eq "$(print -r -- $out | cut -f3,5,6 | tr '\t' ' ')" "2 92 1" \
  "a second hot check makes the process due"
assert_eq "$(detect 1300 $state $WORK/ps2 | awk -F'\t' '$1==103 {print $3,$5}')" "0 1" \
  "a quiet process resets to zero hot runs"
assert_eq "$(detect 1300 $state $WORK/ps2 | awk -F'\t' '$1==104 {print $3,$5}')" "0 0" \
  "a reused pid with a new command starts from zero"

printf 'at\t5000\n102\t100.00\t2\t4900\t90\t/bin/py\n' > $state
printf '  102   6:15.00 /bin/py\n' > $WORK/ps4
assert_eq "$(detect 5300 $state $WORK/ps4 | cut -f3,4,6 | tr '\t' ' ')" "3 4900 0" \
  "an alert inside the cooldown is not due again"
assert_eq "$(detect 8600 $state $WORK/ps4 | cut -f6)" "0" \
  "a long gap (sleep) averages below the threshold"
printf 'at\t8300\n102\t100.00\t2\t4900\t90\t/bin/py\n' > $state
assert_eq "$(detect 8600 $state $WORK/ps4 | cut -f3,6 | tr '\t' ' ')" "3 1" \
  "an alert past the cooldown is due again"

# --- app_for_path ---------------------------------------------------------------

roots=(/r/monorepo /r/monorepo-worktrees/hw-import)
assert_eq "$(app_for_path /r/monorepo/apps/market-data-server/.venv/bin/python3 $roots)" \
  "market-data-server" "labels a path under apps/ with the app name"
assert_eq "$(app_for_path /r/monorepo/libs/ui-kit/x.js $roots)" "ui-kit" \
  "labels a path under libs/ with the lib name"
assert_eq "$(app_for_path /r/monorepo/node_modules/.bin/nx $roots)" "monorepo" \
  "labels other repo paths as monorepo"
assert_eq "$(app_for_path /r/monorepo $roots)" "monorepo" "labels the repo root"
assert_eq "$(app_for_path /r/monorepo-worktrees/hw-import/apps/health-watcher/src $roots)" \
  "health-watcher [worktree hw-import]" "names the worktree"
assert_eq "$(app_for_path /r/monorepo-other/apps/a $roots || print none)" "none" \
  "a sibling directory sharing the prefix is outside the repo"
assert_eq "$(app_for_path /Applications/Chrome.app $roots || print none)" "none" \
  "paths outside the repo do not match"

# --- app_for_process --------------------------------------------------------------

typeset -A fake_cwd
fake_cwd=(201 /r/monorepo/apps/ai-gateway 202 /Users/me)
cwd_of() { print -r -- ${fake_cwd[$1]:-} }

assert_eq "$(app_for_process 200 '/r/monorepo/apps/a/.venv/bin/python3 .venv/bin/uvicorn x' $roots)" \
  "a" "uses an absolute path in the command"
assert_eq "$(app_for_process 200 'node --config=/r/monorepo/apps/b/vite.config.ts' $roots)" \
  "b" "uses a path given as --option=value"
assert_eq "$(app_for_process 201 '/var/folders/x/go-build/exe/main' $roots)" "ai-gateway" \
  "falls back to the working directory"
assert_eq "$(app_for_process 202 '/usr/bin/top' $roots || print none)" "none" \
  "a process outside the repo has no app"

# --- check ---------------------------------------------------------------------

export DEV_CPU_WATCH_STATE_DIR=$WORK/state
fake_now=0
now_epoch() { print -r -- $fake_now }
ps_snapshot() { cat $WORK/ps }
worktree_roots() { print -l -- $roots }
notify() { print -r -- "$1|$2" >> $WORK/notified }
: > $WORK/notified

write_ps() {
  printf '  300 %s /r/monorepo/apps/market-data-server/.venv/bin/python3 .venv/bin/uvicorn src.main:app --reload\n' $1 > $WORK/ps
  printf '  301 %s /Applications/Chrome.app/Contents/MacOS/Chrome\n' $1 >> $WORK/ps
}

fake_now=1000; write_ps 0:10.00; cmd_check > /dev/null
fake_now=1030; write_ps 9:10.00
assert_eq "$(cmd_check)" "dev-cpu-watch: last check was 30s ago, skipping" \
  "skips a check that comes too soon after the last one"
fake_now=1300; write_ps 2:25.00; cmd_check > /dev/null
assert_eq "$(wc -l < $WORK/notified | tr -d ' ')" "0" "one hot check does not alert"
fake_now=1600; write_ps 4:40.00; cmd_check > /dev/null
assert_eq "$(wc -l < $WORK/notified | tr -d ' ')" "1" \
  "two hot checks alert once, and only for the repo process"
assert_eq "$(cat $WORK/notified)" \
  "Dev CPU watch: market-data-server|45% CPU for 10 min (pid 300). /r/monorepo/apps/market-data-server/.venv/bin/python3 .venv/bin/uvicorn src.main:app --reload" \
  "the notification names the app, CPU, duration, pid and command"
assert_eq "$(cut -f2-4 $DEV_CPU_WATCH_STATE_DIR/alerts.log)" $'market-data-server\t45%\t300' \
  "logs the alert"
fake_now=1900; write_ps 6:55.00; cmd_check > /dev/null
assert_eq "$(wc -l < $WORK/notified | tr -d ' ')" "1" "does not repeat inside the cooldown"
assert_eq "$(head -1 $DEV_CPU_WATCH_STATE_DIR/state.tsv)" $'at\t1900' "records the check time"

# --- plist ---------------------------------------------------------------------

plist_content /opt/repo/apps/dev-cpu-watch/src/dev-cpu-watch /opt/repo/tmp/dev-cpu-watch > $WORK/agent.plist
assert_eq "$(plutil -lint $WORK/agent.plist | sed 's/.*: //')" "OK" "the launchd plist is valid"
assert_eq "$(plutil -extract StartInterval raw $WORK/agent.plist)" "300" "runs every 5 minutes"
assert_eq "$(plutil -extract ProgramArguments.1 raw $WORK/agent.plist)" \
  "/opt/repo/apps/dev-cpu-watch/src/dev-cpu-watch" "runs the script from the given path"

print "$passes passed, $fails failed"
(( fails == 0 ))
