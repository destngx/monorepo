# User Stories: Settings & Configuration (Data Integrity)

## 1. Persona Definition: System Administrator

- **Role**: Specialized Wealth Manager focused on backend connection reliability, API keys, privacy levels, and localization.
- **Needs**: Safe API key storage, "Connected" feedback, privacy toggles, and multi-language support.

---

## 2. API & Integration Setup

### US-SET-001: Mission-Critical Google Sheets Connection

- **Story**: As a System Administrator, I want to input my Google Sheet "Spreadsheet ID" and "Account Sync ID," so that the entire dashboard connects to my private data source.
- **Acceptance Criteria**:
  - Configuration for `SPREADSHEET_ID` and `ACCOUNT_ID`.
  - Must include a "Test Connection" button with a green/red status indicator.

### US-SET-002: Binance / P2P Exchange Proxy Setup

- **Story**: As a System Administrator, I want to store my API keys for exchange proxies (e.g., Binance), so that the "Market Alpha" signals can fetch my actual balances in USD.
- **Acceptance Criteria**:
  - API Keys must be masked after input with a "Click to reveal" option.
  - Keys must be stored either in `.env.local` or an encrypted backend field.

---

## 3. Privacy & Safety Controls

### US-SET-003: "Stealth Mode" Global Privacy Toggle

- **Story**: As a System Administrator, I want the application to mask all balances and account numbers by default, with a fast-access "Eye" icon in the header to uncover them temporarily, so that I can safely open the app in public spaces.
- **Acceptance Criteria**:
  - **Masked by Default**: On every initial load and manual refresh, values must be obfuscated (e.g., `••••••`).
  - **Header Toggle**: A persistent "Privacy Toggle" icon must be available in the main Navigation Header.
  - **Navigation Reset**: Each time the user changes pages (Client-side routing), the state must reset to **"Masked"** by default.
  - **Granular Masking**: System must mask Account Names, Bank Institution Names, and Currency Balances.

### US-SET-004: UI "Glassmorphism" Intensity Control

- **Story**: As a System Administrator, I want to adjust the "Backdrop Blur" and "Transparency" of the UI, so that I can optimize the experience for my monitor's readability or performance.
- **Acceptance Criteria**:
  - A slider for "Glassmorphism Strength" (0% to 100%).
  - Effect must be visible on the "Settings" page cards as a live preview.

---

## 4. Globalization & Regional

### US-SET-005: Bilingual Support (EN / VI)

- **Story**: As a System Administrator, I want to toggle the entire app between English and Vietnamese, so that I can use the terminology I am most comfortable with for financial accounts.
- **Acceptance Criteria**:
  - A simple "Vietnam / UK" flag toggle on the settings page.
  - All labels, menus, and AI-generated summaries should translate.

---

## 5. Security & Maintenance

### US-SET-006: System-Wide Data Cache Reset

- **Story**: As a System Administrator, I want a "Clear Cache" button for the Upstash Redis store, so that I can force a fresh fetch from Google Sheets if I notice a data discrepancy.
- **Acceptance Criteria**:
  - A "Flush Cache" command button in the "Advanced" settings section.
  - A warning modal: "This will reload all data from Google Sheets - proceed?"

---

## 6. Connectivity & Performance

### US-SET-007: No-Internet "Full-Stop" Warning

- **Story**: As a System Administrator, I want a persistent "No Connection" modal to appear if my internet drops, so that I don't accidentally try to enter data that will fail to sync.
- **Acceptance Criteria**:
  - Modal must block all interactions until signal returns.
  - No "Offline Retry Queue" required; the priority is preventing stale data entries.

### US-SET-008: External Cache Storage (Snapshots)

- **Story**: As a System Administrator, I want to store "Pre-Calculated Snapshots" of my Net Worth metrics in Redis and LocalStorage, so that I can view my history "on-the-fly" without taxing the Google Sheets API.
- **Acceptance Criteria**:
  - Must use Upstash Redis as the persistent snapshot store.
  - LocalStorage should serve as a secondary cache for the current session.
  - Historical data must be computed from ledger snapshots periodically.
