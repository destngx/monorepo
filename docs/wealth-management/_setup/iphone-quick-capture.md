# iPhone Quick Capture

Send a bank notification or app screen from the iPhone to the engine in one tap (WM-CAP-15..20). The engine stores the screenshot, reads it in the background and turns it into **draft transactions**. Nothing reaches the sheet until you approve a draft on the **Transactions → Drafts** page.

The flow: AssistiveTouch → Shortcut → screenshot → menu (**Send** or **Add note & send**) → `POST /api/capture/inbox` over Tailscale.

## 1. Engine

1. Install Tailscale on the Mac and the iPhone and sign both into the same tailnet. Note the Mac's tailnet address (`tailscale ip -4`, a `100.x.y.z` address) or its MagicDNS name (`<mac>.<tailnet>.ts.net`).
2. Set the API token, once, with the engine still on localhost (the default `HOST`): open `http://localhost:9090/ui`, go to **Settings → General → API token**, press **Generate**, then **Set on engine**. The token applies at once and is saved in that browser too. Press **Copy** and keep it for the Shortcut (step 5 of section 2). Without a token the engine refuses every change once it listens on the tailnet, the token change included, which is why the first one is set on localhost.
3. Restart the engine reachable from the tailnet (bootstrap variables, [README section 3.1](./README.md#31-bootstrap-process-environment)):

   | Variable | Value                                                                                           |
   | -------- | ----------------------------------------------------------------------------------------------- |
   | `HOST`   | The Mac's tailnet address. `0.0.0.0` also works but listens on every network, the LAN included. |
   | `PORT`   | Optional, default `9090`.                                                                       |

   Bound to the tailnet address, the UI is at `http://<tailnet address>:<PORT>/ui`, not `localhost`. That is a new address for the browser, so paste the token under **Settings → General → API token** and press **Save in this browser** (any other browser or device needs the same, once). To change the token later, enter or generate a new one there and press **Set on engine**: the old one stops working at once, so update the Shortcut too.

4. The sheet and AI must be configured: captures are accepted without AI, but stay queued until it is configured.
5. Captures are kept in `capture.db` next to `settings.db` (or `WM_CAPTURE_PATH`), mode `0600`.

Check from the iPhone: open Safari at `http://<tailnet address>:<PORT>/api/health`. A JSON page means the phone can reach the engine.

## 2. Shortcut

In the **Shortcuts** app, tap **+** and name the shortcut **Capture to Wealth**. Add these actions in order (search each name in the action list):

1. **Take Screenshot**. The screen is captured before the menu appears, so the menu never covers it.
2. **Text**: leave it empty.
3. **Set Variable**: name `Note`, input **Text** (the empty text from step 2).
4. **Choose from Menu**, prompt `Wealth`, with three items:
   - **Send**: no actions.
   - **Add note & send**:
     1. **Ask for Input**, type Text, prompt `Note`.
     2. **Set Variable** `Note` to **Provided Input**.
   - **Cancel**: **Stop This Shortcut**.
5. After **End Menu**, **Get Contents of URL**:
   - URL: `http://<tailnet address>:<PORT>/api/capture/inbox`
   - Method: `POST`
   - Headers: `Authorization` = `Bearer <token>` (the word `Bearer`, a space, then the token from the Settings page, typed or pasted as text)
   - Request Body: **Form**, with two fields:
     - `image`, type **File**, value **Screenshot** (from step 1)
     - `note`, type **Text**, value **Note** (the variable)
6. **Get Dictionary Value**: get **Value** for key `id` in **Contents of URL**.
7. **If** **Dictionary Value** **has any value**:
   - **Show Notification**: `Captured`.
   - **Otherwise**: **Show Alert** with **Contents of URL** as the message (the engine's error, e.g. `missing or invalid API token`).
   - **End If**.

The first run asks to allow the shortcut to connect to the engine's address: choose **Always Allow**.

Sending the same screenshot twice is safe: the engine returns the first capture (`duplicate: true`) and creates nothing new. Overlapping screenshots (the same transaction on two screens) become one draft.

## 3. AssistiveTouch

**Settings → Accessibility → Touch → AssistiveTouch**, then either:

- **Custom Actions → Long Press → Capture to Wealth** (recommended: hard to trigger by accident), or
- **Customise Top Level Menu**, tap an icon (or **+**) and pick **Capture to Wealth** under **Shortcuts**.

Open the bank app or notification you want, long-press the AssistiveTouch button, choose **Send** or **Add note & send**.

## 4. Review

The **Drafts** entry under Transactions shows the number of drafts waiting. On the page:

- Edit drafts like Smart Capture rows, then **Approve** writes the selected ones to the sheet; **Reject** discards them. A rejected transaction is not offered again by later screenshots.
- Each draft has one confidence. Only drafts above 80% start selected. Click the confidence to see why: the reading, a sheet row it may duplicate, a missing account or date. A draft that likely duplicates a sheet row gets a low confidence.
- The balances above the list show each account after the selected drafts, so ticking or unticking one shows its effect at once. A balance read from a screenshot shows Matches or Off by.
- When you approve a draft under a different payee than the bank showed (say NGUYEN VAN A as "Banh mi stall"), the engine remembers it and fills that payee in for later captures, marked "Payee learned". The rule is saved in the CaptureRules tab (kind `counterparty`), where you can edit or delete it. Rejecting a draft that is likely already recorded teaches the payee of the sheet row it matched.
- Each draft has an Add note button that opens a "Note for next time" box: write what the payment was for ("banh mi, breakfast"). The note is kept by the app, not written to the sheet, and later drafts from the same person or shop show it and use it to pick the category.
- **Captures** lists what was sent: queued, reading, retrying (1 min, 5 min, 15 min, 1 h, 6 h), done or failed. Failed captures can be retried or discarded.
- **Upload** sends screenshots from the browser the same way, for photos already in the library.

Screenshots and notes are deleted once every draft read from them is decided (immediately when nothing was found). Decided drafts are purged after 90 days.

## 5. Troubleshooting

| Symptom                                             | Cause and fix                                                                                                                                                      |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Alert `missing or invalid API token` / 401          | The `Authorization` header does not match the engine's token. Check `Bearer` (capital B), then one space, then the token; after a token change, paste the new one. |
| 403 `writes are disabled`                           | No API token is set and the engine is bound to a non-loopback `HOST`. Start it on localhost, set the token (section 1, step 2) and restart.                        |
| `Could not connect to the server` or a long timeout | Tailscale is off on one device, the Mac is asleep, the engine is not running, or `HOST` is still `127.0.0.1`.                                                      |
| 400 `send a multipart form...`                      | The request body is JSON, not **Form**. Change Request Body to Form.                                                                                               |
| 400 about the image type or size                    | Only PNG, JPEG, WebP and GIF up to `CAPTURE_MAX_ITEM_MB` (default 5 MB). Take Screenshot produces PNG.                                                             |
| 503 `SHEETS_NOT_CONFIGURED`                         | The sheet is not configured on the engine; quick capture needs it.                                                                                                 |
| Captured, but the draft never appears               | See **Captures** on the Drafts page: the capture may be retrying (AI unreachable) or "Nothing found".                                                              |

Reads (`GET`) need no token, like every other engine route: anyone who can reach the engine can view drafts and stored screenshots. Keep it on the tailnet and off public networks.
