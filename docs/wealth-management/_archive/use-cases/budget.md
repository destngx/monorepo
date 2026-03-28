# Use Case Specification: Budget Control (Monthly Tactical)

## UC-BUD-001: Category-Level Budget Allocation

### 1. Summary

The user sets a new monthly limit for a specific spending category (e.g., "Dining").

### 2. Actors

- **Primary Actor**: Budget Controller (User)
- **Secondary Actor**: Budget API, Google Sheets.

### 3. Preconditions

- User is on the "Budget Control" page.
- Category (e.g., "Dining") already exists in the ledger.

### 4. Basic Flow

1.  User clicks the "Set Target" button for the "Dining" category.
2.  System opens a small popover or input field containing the current target (e.g., `0 VND`).
3.  User enters `5,000,000` (VND).
4.  User clicks "Save Target."
5.  System sends the payload to `/api/budget/upsert`.
6.  System updates the target row in the Google Sheets `Budget` tab.
7.  System recalculates the "Remaining Surplus" and "Percentage Spent" on the Budget Page.

### 5. Postconditions

- The "Dining" budget now reflects the new target, and the Progress Bar updates to show current relative spending.

---

## UC-BUD-002: Mid-Month Budget Pivot (Re-allocation)

### 1. Summary

User notices they overspent in "Food" but have surplus in "Shopping," and moves the surplus to compensate.

### 2. Actors

- **Primary Actor**: Budget Controller (User)

### 3. Preconditions

- Both "Food" and "Shopping" categories have active monthly targets.

### 4. Basic Flow

1.  User views the "Budget Control" dashboard.
2.  User sees "Food" is at 110% (Red) and "Shopping" is at 30% (Green).
3.  User selects "Move Budget" or "Edit Both."
4.  User subtracts `1,000,000` from "Shopping" budget.
5.  User adds `1,000,000` to "Food" budget.
6.  User clicks "Confirm Change."
7.  System performs a "Double Update" locally and sends both changes to the backend.
8.  System refreshes the view: "Food" is now at ~90% (Yellow), and "Shopping" has a lower ceiling.

### 5. Postconditions

- Total monthly budget remains balanced, but individual categories are re-calibrated.

---

## UC-BUD-003: Monthly Burn-Rate Verification (Check-in)

### 1. Summary

User checks their "Estimated Surplus" on the 15th of the month to see if they can afford an extra investment.

### 2. Actors

- **Primary Actor**: Budget Controller (User)

### 3. Basic Flow

1.  User opens the "Budget Control" page.
2.  System calculates: (Total Spending so far) compared to (Total monthly budget).
3.  System displays a "Velocity Metric": "Spending at 1M VND / Day. On track for 30M VND total (Target: 35M VND)."
4.  User sees a "Predicted Surplus" of 5M VND.
5.  User decides to move that 5M VND into their "Securities Account" for investment.

### 4. Postconditions

- User has verified their spending speed and identified surplus capital mid-month.
