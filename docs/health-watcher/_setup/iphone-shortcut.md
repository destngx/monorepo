# iPhone: send scans to Health Watcher

Scan a paper with the iPhone's own document scanner, or pick any PDF or photo, and send it to the Health Watcher inbox from the share sheet (HW-IMP-4). No app install and no developer account needed.

The flow: Notes or Files scanner (or Zalo, Mail, Photos) → Share → **Send to Health** → `POST /api/inbox` over Tailscale → the inbox, where you file it later.

## 1. Engine reachable from the phone

1. Install Tailscale on the Mac and the iPhone and sign both into the same tailnet. Note the Mac's tailnet address (`tailscale ip -4`, a `100.x.y.z` address).
2. On the Mac, open **Settings → Engine settings** (engine admins only), set **Host** to that address and save. Restart the engine:

   ```sh
   pnpm exec nx run health-watcher:start
   ```

3. On the iPhone, open Safari at `http://100.x.y.z:9191/api/health`. A short JSON page means the phone can reach the engine.
4. Optional: open `http://100.x.y.z:9191/ui` in Safari, log in, then **Share → Add to Home Screen**. The app opens full screen from the home screen, and the Inbox page's _Take photos_ button uses the camera.

## 2. Upload token

In Health Watcher on any device: **Settings → iPhone Shortcut**. Name the token (for example `iPhone Shortcut`) and press **Create token**. Two values appear, once:

- the upload URL, `http://100.x.y.z:9191/api/inbox`
- the header value, `Bearer hwu_…`

The token can only add files to this family's inbox. It can't read anything. Revoke it on the same page if the phone is lost. A token stops working when its owner becomes a viewer or leaves the family.

## 3. Shortcut

In the **Shortcuts** app, tap **+**, name it **Send to Health**, then:

1. Tap the **i** (details) button and turn on **Show in Share Sheet**. Under share sheet types keep **PDFs** and **Images** only.
2. Add **Ask for Input**: type Text, prompt `Note (optional)`, allow an empty answer.
3. Add **Repeat with Each** item in **Shortcut Input**. Inside the repeat:
   1. **Get Contents of URL**:
      - URL: the upload URL from step 2
      - Method: `POST`
      - Headers: `Authorization` = the header value from step 2
      - Request Body: **Form**, two fields:
        - `file`, type **File**, value **Repeat Item**
        - `note`, type **Text**, value **Provided Input**
   2. **Get Dictionary Value**: key `files` in **Contents of URL**. Every accepted upload answers with `files`, a duplicate too.
   3. **If** **Dictionary Value** **does not have any value**: **Show Alert** with **Contents of URL** as the message (the engine's error, such as a revoked token or a file type it doesn't accept), then **Stop This Shortcut**. **End If**.
4. After **End Repeat**, add **Show Notification**: `Sent to Health Watcher`.

Each shared file becomes its own inbox item. To keep pages together as one document, scan them as one multi-page PDF in Notes or Files.

## 4. Scanning tips

- **Paper**: the Notes or Files scanner crops and flattens the page and makes a PDF. In Files: **⋯ → Scan Documents**.
- **Record books** (sổ khám bệnh): one photo per open spread, flat, in daylight.
- **Imaging films**: hold the film against a window or a laptop showing a plain white page at full brightness. No flash, the whole film in the frame. File it as _Imaging film_, and file the written radiology report as well, since the report is what holds the findings.
- Files the family already has are recognised by their content and skipped, so sending something twice is harmless.
