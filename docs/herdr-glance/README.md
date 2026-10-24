# herdr-glance

A phone page for the coding agents running in [herdr](https://herdr.dev) on the laptop. It lists every agent with its state, shows an agent's recent output, and lets you answer it with keys or a message. It is built for an iPhone 12 Pro (390 × 844 points, notch and home bar) and can be added to the home screen.

The engine is one Go binary in `apps/herdr-glance` that uses only the standard library. It talks to herdr's socket at `~/.config/herdr/herdr.sock` and serves the built SvelteKit UI at `/ui`. It listens on `127.0.0.1:9393` and on this Mac's Tailscale IP, so your own tailnet devices can reach it and the local network and the internet cannot.

## 1. Use it

Run it in a terminal while you need it:

```sh
pnpm exec nx serve herdr-glance   # build the UI and engine, run until Ctrl+C
```

`serve` builds the engine and UI of the checkout you run it from and starts the engine on `127.0.0.1:9393` and the Tailscale IP from `tailscale ip -4`. It prints the address to open on the phone, such as `http://100.x.y.z:9393/`. It stops if Tailscale is not connected, and refuses to start when something already answers on the port, such as the launchd agent below. macOS may ask once whether `herdr-glance` may accept incoming connections; allow it.

Or keep it running in the background, started at login:

```sh
pnpm exec nx run herdr-glance:load-agent    # build, start at login with launchd
pnpm exec nx run herdr-glance:status        # agent state, engine health, phone address, last log lines
pnpm exec nx run herdr-glance:unload-agent  # stop it and remove the launchd agent
```

`load-agent` builds the engine and the UI in the primary checkout (`~/projects/monorepo`), even when you run it from a worktree. Then it loads `~/Library/LaunchAgents/com.destngx.monorepo.herdr-glance.plist`, which listens on the same two addresses and restarts the engine if it exits. If the Mac starts before Tailscale connects, launchd retries every 10 seconds until the address exists. After pulling `main`, run `load-agent` again to rebuild and restart. The launchd log is in `tmp/herdr-glance/launchd.log` in the primary checkout.

All of these targets call `apps/herdr-glance/scripts/herdr-glance`. Set `HERDR_GLANCE_PORT` to use another port.

On the iPhone, open the address in Safari with Tailscale connected, then choose Share → Add to Home Screen. The page then opens full screen like an app.

For UI work, `nx run herdr-glance:serve-ui` runs Vite on port 5176 and proxies `/api` to a running engine (`HERDR_GLANCE_DEV_ENGINE` overrides `http://127.0.0.1:9393`). `nx run herdr-glance:start` builds the UI and runs the engine with `go run` on `127.0.0.1` only.

## 2. The pages

- **Agents** lists agents grouped by state: Needs you (herdr's `blocked`), Working, Done, Idle, Unknown. A line at the top counts them, and the tab title shows how many need you. A dot says whether the page is getting live updates.
- **One agent** shows its recent output, about one terminal screen, newest at the bottom. herdr cannot read further back while an agent works, because that means scrolling the agent's screen. It reloads every 2 seconds while the agent works or waits for you, every 8 seconds otherwise, and at once when its state changes. The keys `1`, `2`, `3`, Enter, Esc, ↑, ↓, Tab and Shift+Tab answer menus such as Claude Code's permission question. The text box sends a message. herdr refuses a message while the agent waits for an answer, so the page asks you to use the keys first.

herdr's `recent_unwrapped` read gives each terminal line unwrapped, with its ANSI colour codes. The page turns the codes into coloured text (`lib/ansi.ts`), then sorts Claude Code's screen into blocks (`lib/screen.ts`):

- **Your messages** start with `❯` on a background colour. They show as an accent bubble marked "You".
- **Replies** start with a white `⏺` and continue on indented lines. They show in the sans font with a left border, re-flowed to the phone's width, with their inline code colours. A reply whose first line scrolled off the top is recognised by its indented, default-colour text.
- **Tool calls** start with a green, red or grey `⏺`. They, the status line and everything else stay monospace in the agent's colours. Output from other agents is all terminal text.

The output panel stays dark in both themes, because terminal agents pick their colours for a dark terminal. The page drops the side panel Claude Code draws on the right (a scrollbar, a close mark and a menu on one background colour, past column 40), shortens full-width rules to 16 characters, and wraps everything to the phone's width, so nothing scrolls sideways. Claude Code shortens long messages on its own screen, so a long message of yours ends in "…" on the phone too.

## 3. Load on the laptop

While no page is open, the engine does nothing. While a page is open, the engine asks herdr for the agent list every 2 seconds and sends it to the phone only when it changed. The agent page reads that agent's output on the timers above. The page closes its stream when the phone locks or switches apps, so the polling stops with it.

## 4. API

| Method and path                  | Body               | Reply                                                        |
| -------------------------------- | ------------------ | ------------------------------------------------------------ |
| `GET /api/agents`                |                    | `{agents: Agent[]}`                                          |
| `GET /api/events`                |                    | Server-sent events, one `{agents, error?, code?}` per change |
| `GET /api/agents/{pane}/screen`  |                    | `{text}`: about one terminal screen, plain text              |
| `POST /api/agents/{pane}/prompt` | `{text}`           | 204                                                          |
| `POST /api/agents/{pane}/keys`   | `{keys: string[]}` | 204                                                          |
| `GET /healthz`                   |                    | 204                                                          |

Errors are `{error, code}`. `HERDR_UNREACHABLE` (503) means herdr is not running. `AGENT_BLOCKED` (409) means the agent waits for an answer. `NOT_FOUND` (404) means the pane is gone.

Every herdr call is in `herdr/client.go`. A herdr release that changes the socket API should only need changes there.

## 5. Security

Anyone who can reach the page can type into your agents, so the engine checks every request:

- It listens on `127.0.0.1` and the Tailscale IP only, never `0.0.0.0`. A device on the same Wi-Fi cannot connect. Traffic to the Tailscale IP travels inside Tailscale's WireGuard tunnel, so it is encrypted even though the page is plain `http://`.
- It answers only to `localhost`, IP addresses and `*.ts.net` host names, plus any in `HERDR_GLANCE_ALLOWED_HOSTS`. That stops DNS rebinding, which needs a host name.
- A `POST` must come from the page itself. The engine checks `Origin` against the host, refuses `Sec-Fetch-Site` other than `same-origin`, and accepts only JSON bodies. Another site open in a browser on the laptop cannot post a prompt.
- Keys are limited to the list above. `ctrl+c` and other control keys are refused. A message is at most 8,000 characters.

Every device on your tailnet can open the page. To limit it to the phone, add a Tailscale access rule for port 9393.

## 6. Settings

| Variable                     | Default                      | Meaning                                              |
| ---------------------------- | ---------------------------- | ---------------------------------------------------- |
| `HERDR_GLANCE_ADDR`          | `127.0.0.1:9393`             | Listen addresses, comma separated                    |
| `HERDR_GLANCE_UI_DIR`        | `ui/build`                   | Built UI folder                                      |
| `HERDR_GLANCE_ALLOWED_HOSTS` |                              | Extra host names, comma separated                    |
| `HERDR_SOCKET_PATH`          | `~/.config/herdr/herdr.sock` | herdr's socket                                       |
| `HERDR_SESSION`              |                              | A named herdr session, used if no socket path is set |
| `HERDR_GLANCE_PORT`          | `9393`                       | `serve` and `load-agent`: port on both addresses     |
