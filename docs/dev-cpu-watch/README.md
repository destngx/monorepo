# Dev CPU watch

A launchd agent that checks CPU use every 5 minutes and sends a macOS notification when a process from this repo, or one of its worktrees, keeps the CPU busy. It ignores every process outside the repo.

It exists because of one incident. `nx serve market-data-server` ran `uvicorn --reload` without `watchfiles` installed, so uvicorn fell back to polling. It ran `stat()` on about 4,500 files in `.venv` several times a second and held 45% of a core for a day and a half before anyone noticed. Commit `4727f722` fixed that cause. This tool catches the next runaway process of any kind.

## 1. Use it

```sh
pnpm exec nx run dev-cpu-watch:load-agent    # load the launchd agent; it runs a check at once, then every 5 minutes
pnpm exec nx run dev-cpu-watch:status        # agent state, last check, processes at or above the threshold, recent alerts
pnpm exec nx run dev-cpu-watch:check         # one check by hand
pnpm exec nx run dev-cpu-watch:unload-agent  # stop it and remove the plist
```

The Nx targets call `apps/dev-cpu-watch/src/dev-cpu-watch` with `install`, `status`, `check` and `uninstall`. You can also run the script directly.

`load-agent` writes `~/Library/LaunchAgents/com.destngx.monorepo.dev-cpu-watch.plist`. The plist points at the script in the primary checkout (`~/projects/monorepo`), even when you run `load-agent` from a worktree. Pulling `main` updates the agent with no reinstall. Run `load-agent` again only after changing `DEV_CPU_WATCH_INTERVAL`, which goes into the plist.

The notification title is `Dev CPU watch: <app>`. The text gives the CPU percentage, how long it has stayed high, the pid and the command, for example:

```text
Dev CPU watch: market-data-server
45% CPU for 10 min (pid 74965). ~/projects/monorepo/apps/market-data-server/.venv/bin/python3 .venv/bin/uvicorn src.main:app --reload --port 8000
```

`osascript` posts the notification, so macOS lists it under Script Editor. If no notification appears, check System Settings → Notifications → Script Editor. Every alert is also appended to `tmp/dev-cpu-watch/alerts.log`, so a missed notification is still on record.

## 2. How a check works

1. `ps -Ao pid=,time=,command=` gives each process's total CPU time.
2. `src/detect.awk` subtracts the CPU time saved by the last check and divides by the seconds between the checks. That gives the average CPU over the interval, where 100% is one full core. A short spike, such as a build, averages out.
3. A process at or above the threshold has its hot count raised by one. A process below it goes back to zero. A process counts as the same one only when both its pid and its command match, so a reused pid starts from zero.
4. When the hot count reaches the limit and the process has not alerted within the cooldown, the script looks up who owns it. The first absolute path in the command that is inside the repo decides it, and the working directory from `lsof` is the fallback. Go binaries built by `go run` live in a temp folder, so the working directory is how those are found.
5. A path under `apps/<name>` or `libs/<name>` is labelled `<name>`, and any other repo path is labelled `monorepo`. A worktree adds `[worktree <folder>]`. The repo roots come from `git worktree list`, so a new worktree is covered with no setup.
6. A repo process sends the notification and is logged. A process outside the repo is only marked as checked, which keeps `lsof` to once per cooldown for it.

A check that comes less than 60 seconds after the last one is skipped, because a short interval gives noisy numbers. After the Mac sleeps, the long gap averages the CPU down, so waking up never triggers an alert.

## 3. Settings

The script reads these environment variables and falls back to the defaults below. For the launchd agent, change the defaults at the top of `src/dev-cpu-watch`.

| Variable                     | Default                       | Meaning                                                    |
| ---------------------------- | ----------------------------- | ---------------------------------------------------------- |
| `DEV_CPU_WATCH_THRESHOLD`    | 25                            | Percent of one core, averaged over a check interval        |
| `DEV_CPU_WATCH_RUNS`         | 2                             | Consecutive hot checks before an alert (about 10 minutes)  |
| `DEV_CPU_WATCH_COOLDOWN`     | 3600                          | Seconds before the same process alerts again               |
| `DEV_CPU_WATCH_MIN_INTERVAL` | 60                            | A check sooner than this after the last one is skipped     |
| `DEV_CPU_WATCH_INTERVAL`     | 300                           | launchd `StartInterval`, written into the plist on install |
| `DEV_CPU_WATCH_STATE_DIR`    | `<primary>/tmp/dev-cpu-watch` | State, alert log and launchd log                           |

The state folder holds `state.tsv` (the last check's per-process CPU times and hot counts), `alerts.log` and `launchd.log` (the agent's stdout and stderr). It is inside the repo's ignored `tmp/` folder.

## 4. Why a zsh script

Everything it needs ships with macOS: `/bin/zsh`, `awk`, `ps`, `lsof`, `osascript`, `plutil` and `launchctl`. There is nothing to install and no build step. No runtime path can go stale under launchd, as a mise-managed Node path would after a Node upgrade. The script sets `PATH` to the system folders, so it runs the BSD tools even from a shell that puts GNU tools first.

Rust or Go would read CPU time and working directories from the kernel without parsing `ps` and `lsof` text. That gain is small for a check this size. Rust would also be the first Rust project in the repo, with a toolchain, crates and a rebuild before each install to maintain.

## 5. Tests

```sh
pnpm exec nx run dev-cpu-watch:test   # zsh test/run-tests.zsh
pnpm exec nx run dev-cpu-watch:lint   # zsh -n on both scripts and an awk parse
```

The tests source the script and replace `ps`, `lsof`, the clock, `git worktree list` and the notification with fakes. They cover `ps` time formats, the CPU averaging, hot counts, the cooldown, reused pids, sleep gaps, app labels for apps, libs, worktrees and paths outside the repo, a full run of checks from the first sample to a single alert, and a valid plist.
