# Use Case Specification: Transaction Ledger (The Financial Heartbeat)

## UC-TX-001: Manual Transaction Entry (The High-Friction Action)

### 1. Summary

The user records a new expense or income transaction from their bank statement or following a cash purchase.

### 2. Actors

- **Primary Actor**: Financial Auditor (User)
- **Supporting System**: Transactions API, Google Sheets.

### 3. Preconditions

- User has access to the "Add Transaction" modal.
- System is connected to the backend Google Sheet.

### 4. Basic Flow (Success Scenario)

1.  User clicks the "Add Transaction" button.
2.  System opens a modal with a "Date" (Today), "Amount," "Account," "Note," and "Category."
3.  User enters `1,500,000` (VND).
4.  User selects "Techcombank" from the "Account" dropdown.
5.  User enters "Electricity Bill" in the "Note."
6.  System auto-suggests "Utilities" as the category.
7.  User accepts the suggestion or changes it to "Home."
8.  User clicks "Save."
9.  System validates all fields (e.g., Amount must be positive).
10. System sends the payload via `/api/transactions/upsert`.
11. System adds a new row to the Google Sheets `Transactions` tab.
12. System returns a Success message and closes the modal.

### 5. Alternative/Exception Flows

- **API Failure (E1)**: If the backend is down, system shows a Red alert: "Failed to sync. Transaction saved locally for retry."
- **Duplicate Entry (E2)**: If a transaction with the same Date, Amount, and Note already exists, system warns: "Possible duplicate detected. Proceed anyway?"

---

## UC-TX-002: CSV Bank Statement Import (The Bulk Action)

### 1. Summary

User imports a CSV from Vietcombank or other institutions to update their records at once.

### 2. Actors

- **Primary Actor**: Financial Auditor (User)
- **Supporting System**: CSV Parsing Engine, Transactions API.

### 3. Basic Flow

1.  User clicks the "Import CSV" button.
2.  User selects a CSV file from their local machine.
3.  System displays a "Column Mapper" (e.g., Date -> CSV Column 1, Amount -> Column 4).
4.  System parses the rows and displays a "Preview Table."
5.  System highlights rows that have missing categories.
6.  User types a note/category for the highlighted rows.
7.  User clicks "Import All."
8.  System batch-processes each row to the backend Google Sheet.
9.  System reports: "Successfully imported 45 transactions."

---

## UC-TX-003: Reconciliation Audit (Search & Filter)

### 1. Summary

User searches for a specific expense from 6 months ago to verify a payment.

### 2. Actors

- **Primary Actor**: Financial Auditor (User)

### 3. Basic Flow

1.  User navigates to the "Transaction Ledger" view.
2.  User enters "Grab" into the "Search" field.
3.  System filters the list to show all matching rows.
4.  User adjusts the "Date Range" filter to "January 2024 to February 2024."
5.  User identifies the specific transaction of 45,000 VND on Jan 15th.
6.  User clicks "Expand" to see the "Account Used" (e.g., Vietcombank).

### 4. Postconditions

- User has found the target information and potentially updated the category/note if a discrepancy was found.
