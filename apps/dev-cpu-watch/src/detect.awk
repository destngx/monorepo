# Compares a `ps -Ao pid=,time=,command=` snapshot with the previous state and
# prints the new state, one process per line, tab-separated:
#
#   pid  cpu_seconds  hot_runs  last_due  pct  due  command
#
# pct is the process's average CPU over the interval since the last check (100 =
# one full core). hot_runs counts consecutive checks at or above the threshold.
# due is 1 when hot_runs reached `runs` and the last alert for the process is at
# least `cooldown` seconds old.
#
# Usage:
#   awk -v now=EPOCH -v threshold=25 -v runs=2 -v cooldown=3600 \
#     -f detect.awk STATE_FILE PS_SNAPSHOT
#
# STATE_FILE is a previous output with an "at<TAB>epoch" first line, or
# /dev/null on the first run. A process is matched by pid and command, so a
# reused pid starts from zero.

function seconds(t,   days, dash, parts, n, i, s) {
  days = 0
  dash = index(t, "-")
  if (dash) {
    days = substr(t, 1, dash - 1)
    t = substr(t, dash + 1)
  }
  n = split(t, parts, ":")
  s = 0
  for (i = 1; i <= n; i++) s = s * 60 + parts[i]
  return days * 86400 + s
}

FILENAME == ARGV[1] {
  n = split($0, f, "\t")
  if (FNR == 1 && f[1] == "at") {
    prev_at = f[2]
    next
  }
  if (n < 6) next
  key = f[1] "\t" f[6]
  prev_cpu[key] = f[2]
  prev_hot[key] = f[3]
  prev_last[key] = f[4]
  next
}

{
  line = $0
  sub(/^[ \t]+/, "", line)
  sp = index(line, " ")
  if (!sp) next
  pid = substr(line, 1, sp - 1)
  rest = substr(line, sp + 1)
  sub(/^ +/, "", rest)
  sp = index(rest, " ")
  if (!sp) next
  cpu = seconds(substr(rest, 1, sp - 1))
  cmd = substr(rest, sp + 1)
  gsub(/\t/, " ", cmd)

  key = pid "\t" cmd
  interval = now - prev_at
  pct = 0
  hot = 0
  last = 0
  if (prev_at > 0 && interval > 0 && (key in prev_cpu)) {
    last = prev_last[key]
    if (cpu >= prev_cpu[key]) pct = int((cpu - prev_cpu[key]) / interval * 100 + 0.5)
    if (pct >= threshold) hot = prev_hot[key] + 1
  }
  due = (hot >= runs && (last == 0 || now - last >= cooldown)) ? 1 : 0
  printf "%s\t%.2f\t%d\t%d\t%d\t%d\t%s\n", pid, cpu, hot, last, pct, due, cmd
}
