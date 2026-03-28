# Use Case Specification: Dashboard Home (Mission Control)

## UC-HOME-001: Daily Wealth Audit (The Dashboard Login)

### 1. Summary

The user logs in to the dashboard to perform a high-level audit of their financial status at the start of the day.

### 2. Actors

- **Primary Actor**: Wealth Manager (User)
- **Secondary Actor**: Next.js App, Intelligence API, Google Sheets API.

### 3. Preconditions

- User has a connected Google Sheets instance or enough data to calculate net worth.
- System is authenticated and can fetch API data.

### 4. Basic Flow (Success Scenario)

1.  User enters the dashboard via `wealth-management.monorepo.io/`.
2.  System initiates a parallel fetch of `Net Worth` (Accounts) and `Cash Flow Velocity` (Transactions).
3.  System displays the **Total Net Worth** in a large, prominent header.
4.  System displays the **30-day percentage growth/decline**.
5.  System queries the **Intelligence Briefing** API for the "Morning Summary."
6.  System renders the 3-bullet AI briefing at the top of the Home layout.
7.  User scans the current "Capital Surplus."
8.  System shows a visual summary (e.g., Progress Bar) of the month's spending burn rate.

### 5. Alternative/Exception Flows

- **Data Latency (E1)**: If Google Sheets takes > 5s, system shows a "Cached" version or a clear loading state ("Synchronizing Ledger...").
- **No Data (E2)**: If no accounts are found, system displays a "Welcome & Get Started" call-to-action instead of empty dashboards.

### 6. Postconditions

- User is fully informed of their immediate financial status and recent AI-detected anomalies.

---

## UC-HOME-002: Privacy Guard (Public Viewing Mode)

### 1. Summary

User toggles the privacy masking to hide exact currency values while screen-sharing or in a public space.

### 2. Actors

- **Primary Actor**: Wealth Manager (User)

### 3. Preconditions

- Dashboard is open and displaying monetary values.

### 4. Basic Flow

1.  User clicks the "Hide Values" (Eye icon) button on the main header.
2.  System triggers a global UI state change (`isMasked = true`).
3.  All `MaskedBalance` components instantly replace numerical text with `***,*** VND`.
4.  User can navigation between pages (Accounts, Investments) while the masking remains persistent.
5.  User clicks the icon again to "Reveal Values."

### 5. Postconditions

- All sensitive financial metrics are obscured/revealed according to the user's toggle state.

---

## UC-HOME-003: High-Frequency Transaction Entry (Quick Entry)

### 1. Summary

User records a manual expense or income directly from the Dashboard Home to minimize friction.

### 2. Actors

- **Primary Actor**: Wealth Manager (User)
- **Supporting System**: Transaction Ledger API.

### 3. Basic Flow

1.  User clicks the "Add Transaction" shortcut button.
2.  System opens a lightweight "Quick Entry" modal.
3.  User enters Amount, Date (defaults to Today), Description, and Category.
4.  User clicks "Save."
5.  System sends the payload to `/api/transactions`.
6.  System updates the "Monthly Velocity" and "Recent Activity" on the Home page in real-time.
7.  Modal closes with a success indicator.

### 4. Postconditions

- The transaction is successfully recorded in the ledger and all Home page metrics reflect the update.
