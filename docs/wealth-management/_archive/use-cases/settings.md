# Use Case Specification: Settings & Configuration (Data Integrity)

## UC-SET-001: Mission-Critical Ledger Connection (New Setup)

### 1. Summary

The user connects the dashboard app to a NEW Google Sheets repository.

### 2. Actors

- **Primary Actor**: System Administrator (User)
- **Secondary Actor**: Google Sheets API Backend.

### 3. Preconditions

- System is running correctly in the local monorepo environment.

### 4. Basic Flow

1.  User clicks the "Settings" (Gear icon) on the sidebar.
2.  User navigates to the "Integrations" section.
3.  User enters `1A2B3C...` (Spreadsheet ID) into the text field.
4.  User enters `Sheet ID: 0` into the sub-field.
5.  User clicks "Test Connection."
6.  System performs a sample `GET` to the Google Sheets API for the specified ID.
7.  System reports: "Success: Spreadsheet 'WealthManagement_Core' Connected."
8.  User clicks "Save Changes."

### 5. Alternative/Exception Flows

- **Invalid ID (E1)**: If the spreadsheet ID is 404, system shows a Red alert: "Check your ID and ensure the service account has 'Editor' permissions."

---

## UC-SET-002: Privacy & Glassmorphism Personalization (Aesthetic Audit)

### 1. Summary

User adjusts the UI transparency to match their high-resolution monitor's readability profile.

### 2. Actors

- **Primary Actor**: System Administrator (User)
- **Supporting System**: Tailwind CSS Theme Engine.

### 3. Basic Flow

1.  User clicks "Settings" -> "UI Customization."
2.  User sees a "Glassmorphism Strength" slider (Currently 50%).
3.  User moves the slider to 80% (Higher Blur).
4.  System instantly updates the `backdrop-blur` CSS variable across the current page.
5.  User toggles the "Dark Mode / Light Mode" switch to see the contrast.
6.  User clicks "Apply to All Pages."
7.  System saves the configuration to the browser's `LocalStorage`.

### 4. Postconditions

- The UI retains the 80% blur profile throughout the session across all dashboard sub-terminals.

---

## UC-SET-003: Regional Toggle (VND to USD Base)

### 1. Summary

A user living in the US but with VN-based holdings wants to see their "Total Net Worth" in USD as the base currency.

### 2. Actors

- **Primary Actor**: System Administrator (User)

### 3. Basic Flow

1.  User opens "Settings" -> "Regional."
2.  User selects "Base Currency: USD."
3.  User clicks "Apply."
4.  System triggers a global re-calculation of all aggregate KPIs (Net Worth, Budget, Velocity).
5.  System uses the `Fmarket Proxy API` to _divide_ all VND sums by the current rate (e.g., 24,500).
6.  User navigates to the "Dashboard Home."
7.  User sees "Total Net Worth: $102,450.00" instead of the VND equivalent.
8.  User is satisfied with the localized view.

### 4. Postconditions

- All high-level dashboard summaries are expressed in USD while the underlying ledger remains in its original recording currency.
