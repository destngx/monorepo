# User Stories: Transaction Ledger (The Financial Heartbeat)

## 1. Persona Definition: Financial Auditor

- **Role**: Specialized Wealth Manager focused on data entry, bookkeeping accuracy, and expense categorization.
- **Needs**: Fast entry, precise categorization, bulk operations, and searchability.

---

## 2. Data Entry & accuracy

### US-TX-001: Integrated Manual Entry

- **Story**: As a Financial Auditor, I want a fast-entry modal for new transactions, so that I can record an expense (e.g., Coffee) within 15 seconds of the purchase.
- **Acceptance Criteria**:
  - Pre-fills today's date and the last used account by default.
  - Includes: Amount, Date, Category, Account, and Note.
  - Must provide "Save & Add Another" option for batch entry.

### US-TX-002: Multi-Currency Support (USD/VND)

- **Story**: As a Financial Auditor, I want to record transactions in USD (e.g., Stock Buy on NYSE), so that I don't have to manually calculate the VND equivalent at the time of entry.
- **Acceptance Criteria**:
  - Selection of USD or VND in the "Amount" field.
  - System automatically fetches the current spot rate and records both values.

---

## 3. Automation & AI

### US-TX-003: AI-Assisted Smart Categorization

- **Story**: As a Financial Auditor, I want the system to suggest a "Category" when I type a common vendor (e.g., "Grab"), so that I don't have to manually select "Transport" every time.
- **Acceptance Criteria**:
  - Suggestions should appear as the user types in the "Description/Note" field.
  - Suggestions should be based on previous history and global patterns (e.g., Shopee -> Shopping).

---

## 4. Search & Multi-Selection

### US-TX-004: Omni-Search Ledger Filter

- **Story**: As a Financial Auditor, I want to search my entire history by a keyword (e.g., "Apple"), so that I can see every transaction related to that vendor or product across all accounts.
- **Acceptance Criteria**:
  - Filter should apply to Note, Category, and Account fields simultaneously.
  - Results must render instantly (debounced search).

### US-TX-005: Bulk Category Updates

- **Story**: As a Financial Auditor, I want to select 10 transactions and change their category to "Business Expenses" in one click, so that I can fix errors in bulk after a bank import.
- **Acceptance Criteria**:
  - Selectable checkboxes on each transaction row.
  - "Batch Actions" toolbar appears upon selection.

---

## 5. Security & Persistence

### US-TX-006: Direct Google Sheets Persistence

- **Story**: As a Financial Auditor, I want to know that my transaction is saved directly to my private Google Sheet, so that I always have a copy of my records outside of the dashboard app.
- **Acceptance Criteria**:
  - Visual "Saved" indicator after each entry.
  - System must guarantee "At-Least-Once" delivery to the spreadsheet.

---

## 6. Friction Reduction (Hybrid Entry)

### US-TX-007: Sheet-Driven Email Transaction Parser

- **Story**: As a Financial Auditor, I want the system to fetch parsed transaction data from the `EmailNotifications` sheet tab (populated by an external service), so that I can "Review and Approve" pending entries that haven't been processed yet.
- **Acceptance Criteria**:
  - Must only fetch rows where the **Status** column is NOT equal to "DONE" (e.g., "pending" or empty).
  - Must present these rows in a "Review & Approve" queue with extracted Date, Amount, and Vendor.
  - Clicking "Approve" must: (1) Add the transaction to the Ledger and (2) Update the source row status to "DONE".

### US-TX-008: Daily Recap & Entry Reminder

- **Story**: As a Financial Auditor, I want a daily notification (Email/Push) at my preferred time (e.g., 9:00 PM), so that I never forget to reconcile my daily spending while the memory is fresh.
- **Acceptance Criteria**:
  - Configurable notification time in Settings.
  - Notification should include a summary of "Uncategorized" or "Draft" transactions from the parser.
  - Must provide a direct link to the "Quick Entry" modal.

### US-TX-009: Automated Yearly Tab Sharding

- **Story**: As a Financial Auditor, I want the system to automatically store transactions in yearly tabs (e.g., `Transactions_2026`), so that the Google Sheet remains performant and acts like a sharded database.
- **Acceptance Criteria**:
  - The app must dynamically determine the target sheet tab based on the Transaction Date.
  - If the tab for the current year does not exist, the system should prompt for manual creation or fallback to a default.
  - Performance: Ledger view must seamlessly "Union" the current and previous year's data into a single UI list.
