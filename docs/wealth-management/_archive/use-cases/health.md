# Use Case Specification: Financial Health (The Physical)

## UC-HEA-001: Holistic Financial Health Diagnostic (Monthly Audit)

### 1. Summary

The user performs their monthly health check to see if their wealth robustness score has improved.

### 2. Actors

- **Primary Actor**: Resilient Saver (User)
- **Secondary Actor**: Health API, Accounts API, Transactions API.

### 3. Preconditions

- User has at least 3 months of transaction history for an accurate "Average Monthly Expense" calculation.

### 4. Basic Flow

1.  User clicks the "Physical" (Health) tab.
2.  System initiates parallel fetches of "All Balances" and "L6M (Last 6 Months) Transactions."
3.  System calculates:
    - **Total Liquidity**: (Sum of all Cash/Bank accounts).
    - **Avg. Monthly Expenses**: (Sum of all Expense transactions / # of Months).
4.  System displays "Emergency Fund: 7.2 Months" in a Green badge.
5.  System computes the **Holistic Health Score** (e.g., 84/100).
6.  System renders the Large Gauge showing the 84.
7.  User scrolls to the "Radar Chart" to identify weaknesses.
8.  System displays that the "Diversification" point is low (too much concentration in FPT).
9.  User identifies "Doctor's Orders" suggesting they "Trim individual stock holdings by 10% and move to Bonds."

### 5. Alternative/Exception Flows

- **New User (E1)**: If < 1 month of data, system shows "Diagnostic Incomplete" with a list of missing assets (e.g., "Add at least 1 month of expenses to calculate coverage").

---

## UC-HEA-002: "What-If" Debt Simulation (Preparing for a Mortgage)

### 1. Summary

User wants to see how taking on a 30M VND monthly mortgage would affect their "Debt-to-Income (DTI)" and overall health score.

### 2. Actors

- **Primary Actor**: Resilient Saver (User)

### 3. Basic Flow

1.  User views the "DTI Dashboard" section on the Health page.
2.  System shows current DTI: 5% (Very Healthy).
3.  User clicks "Simulate New Debt."
4.  User enters `30,000,000` (VND) as a "New Monthly Payment."
5.  System re-calculates the DTI: (Current Debt + 30M) / Total Income.
6.  System displays "New DTI: 42% (Vulnerable - Above 35% Threshold)."
7.  System lowers the "Holistic Score" by 25 points in the simulation.
8.  User realizes they should wait for a promotion or save a larger down payment.

### 4. Postconditions

- User has a clear data-driven understanding of the impact of major financial decisions on their resilience score.

---

## UC-HEA-003: "Show Your Math" Verification (The Auditor Check)

### 1. Summary

User wants to verify the 7.2 months calculation to ensure the "Avg Monthly Expense" is not skewed by a one-time large purchase.

### 2. Actors

- **Primary Actor**: Resilient Saver (User)

### 3. Basic Flow

1.  User hovers over the "Emergency Fund Coverage" number.
2.  User clicks the "Info" icon.
3.  System opens a popover showing:
    - **Total Cash**: 500,000,000 VND.
    - **Avg. Monthly Expenses**: 69,444,444 VND.
4.  User sees a list of the 6 months used for the average:
    - Jan: 60M, Feb: 65M, Mar: 120M (Large payment!), Apr: 55M, ...
5.  User realizes the 120M from March (New Computer) is skewing the average.
6.  User clicks "Exclude March from average."
7.  System re-calculates: Avg Expense = 60M. New Coverage = 8.3 Months.
8.  User is satisfied with the more "Regular" health score.

### 4. Postconditions

- User has adjusted the diagnostic logic to better reflect their "Standard" financial lifestyle.
