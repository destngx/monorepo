# herdr-mobile

A phone page for the coding agents running in [herdr](https://herdr.dev) on the laptop. It lists every agent with its state, shows an agent's recent output, and lets you answer it with keys or a message. It is built for an iPhone 12 Pro (390 × 844 points, notch and home bar) and can be added to the home screen.

The engine is one Go binary in `apps/herdr-mobile` that uses only the standard library. It talks to herdr's socket at `~/.config/herdr/herdr.sock` and serves the built SvelteKit UI at `/ui`. It listens on `127.0.0.1:9393` only. `tailscale serve` publishes it to your tailnet over HTTPS, so your own devices can reach it and the internet cannot.

## 1. Use it

```sh
pnpm exec nx run herdr-mobile:load-agent    # build, start at login with launchd, publish on the tailnet
pnpm exec nx run herdr-mobile:status        # agent state, engine health, tailnet URL, last log lines
pnpm exec nx run herdr-mobile:unload-agent  # stop serving on the tailnet and remove the launchd agent
```

`load-agent` builds the engine and the UI in the primary checkout (`~/projects/monorepo`), even when you run it from a worktree. Then it loads `~/Library/LaunchAgents/com.destngx.monorepo.herdr-mobile.plist`, which restarts the engine if it exits, and runs `tailscale serve --bg --https=9393 http://127.0.0.1:9393`. It prints the address to open on the phone: `https://<this mac>.<tailnet>.ts.net:9393/`. After pulling `main`, run `load-agent` again to rebuild and restart. The launchd log is in `tmp/herdr-mobile/launchd.log` in the primary checkout.

On the iPhone, open the address in Safari with Tailscale connected, then choose Share → Add to Home Screen. The page then opens full screen like an app.

For development, `nx run herdr-mobile:start` builds the UI and runs the engine. `nx run herdr-mobile:serve-ui` runs Vite on port 5176 and proxies `/api` to the engine (`HM_DEV_ENGINE` overrides `http://127.0.0.1:9393`).

## 2. The pages

- **Agents** lists agents grouped by state: Needs you (herdr's `blocked`), Working, Done, Idle, Unknown. A line at the top counts them, and the tab title shows how many need you. A dot says whether the page is getting live updates.
- **One agent** shows the last 400 lines of output, newest at the bottom. It reloads every 2 seconds while the agent works or waits for you, every 8 seconds otherwise, and at once when its state changes. The keys `1`, `2`, `3`, Enter, Esc, ↑, ↓, Tab and Shift+Tab answer menus such as Claude Code's permission question. The text box sends a message. herdr refuses a message while the agent waits for an answer, so the page asks you to use the keys first.

herdr's `recent_unwrapped` read gives each terminal line unwrapped. The page wraps it to the phone's width and shortens full-width rules to 16 characters (`lib/screen.ts`), so output never scrolls sideways.

## 3. Load on the laptop

While no page is open, the engine does nothing. While a page is open, the engine asks herdr for the agent list every 2 seconds and sends it to the phone only when it changed. The agent page reads that agent's output on the timers above. The page closes its stream when the phone locks or switches apps, so the polling stops with it.

## 4. API

| Method and path                  | Body               | Reply                                                        |
| -------------------------------- | ------------------ | ------------------------------------------------------------ |
| `GET /api/agents`                |                    | `{agents: Agent[]}`                                          |
| `GET /api/events`                |                    | Server-sent events, one `{agents, error?, code?}` per change |
| `GET /api/agents/{pane}/screen`  |                    | `{text}`: the last 400 lines, plain text                     |
| `POST /api/agents/{pane}/prompt` | `{text}`           | 204                                                          |
| `POST /api/agents/{pane}/keys`   | `{keys: string[]}` | 204                                                          |
| `GET /healthz`                   |                    | 204                                                          |

Errors are `{error, code}`. `HERDR_UNREACHABLE` (503) means herdr is not running. `AGENT_BLOCKED` (409) means the agent waits for an answer. `NOT_FOUND` (404) means the pane is gone.

Every herdr call is in `herdr/client.go`. A herdr release that changes the socket API should only need changes there.

## 5. Security

Anyone who can reach the page can type into your agents, so the engine checks every request:

- It listens on `127.0.0.1` only. The phone reaches it through `tailscale serve`, which accepts connections from your tailnet and never from the internet (that would be `tailscale funnel`).
- It answers only to `localhost`, `127.0.0.1`, `::1` and `*.ts.net` host names, plus any in `HM_ALLOWED_HOSTS`. That stops DNS rebinding.
- A `POST` must come from the page itself. The engine checks `Origin` against the host, refuses `Sec-Fetch-Site` other than `same-origin`, and accepts only JSON bodies. Another site open in a browser on the laptop cannot post a prompt.
- Keys are limited to the list above. `ctrl+c` and other control keys are refused. A message is at most 8,000 characters.

Every device on your tailnet can open the page. To limit it to the phone, add a Tailscale access rule for port 9393.

## 6. Settings

| Variable            | Default                      | Meaning                                              |
| ------------------- | ---------------------------- | ---------------------------------------------------- |
| `HM_ADDR`           | `127.0.0.1:9393`             | Listen address                                       |
| `HM_UI_DIR`         | `ui/build`                   | Built UI folder                                      |
| `HM_ALLOWED_HOSTS`  |                              | Extra host names, comma separated                    |
| `HERDR_SOCKET_PATH` | `~/.config/herdr/herdr.sock` | herdr's socket                                       |
| `HERDR_SESSION`     |                              | A named herdr session, used if no socket path is set |
| `HM_PORT`           | `9393`                       | `load-agent` only: local and tailnet port            |
