# User Stories: Budget Control (Monthly Tactical)

## 1. Persona Definition: Budget Controller

- **Role**: Specialized Wealth Manager focused on spending limits, cash-flow discipline, and savings targets.
- **Needs**: Comparative view (Target vs. Actual), progress tracking, "Burn Rate" analysis, and flexible adjustments.

---

## 2. Planning & Alignment

### US-BUD-001: Monthly Category Allocation Targets

- **Story**: As a Budget Controller, I want to define a monthly spending target for each category (e.g., Food: 10M VND), so that I can set a ceiling on my discretionary spending before the month begins.
- **Acceptance Criteria**:
  - Targets must be editable in the Budget view.
  - Changes must persist to the Google Sheets `Budget` tab.

### US-BUD-002: Dynamic Net Savings Goal Indicator

- **Story**: As a Budget Controller, I want to see how my total monthly budget relates to my total income (e.g., "70% of income budgeted"), so that I can ensure I am on track for my 30% savings rate goal.
- **Acceptance Criteria**:
  - Header must subtract "Total Budgeted" from "Average Monthly Income."
  - System must show a "Savings Potential" amount based on current budget limits.

---

## 3. Progress Tracking & Warnings

### US-BUD-003: Visual Health Progress Gauges

- **Story**: As a Budget Controller, I want a progress bar for each category showing how much I've already spent, so that I can see if I'm "running out of room" mid-month.
- **Acceptance Criteria**:
  - Bars must update in real-time as transactions are added to the ledger.
  - Must use color coding (Green: < 75%, Yellow: 75-90%, Red: > 90%).

### US-BUD-004: Estimated Budget Burn Rate

- **Story**: As a Budget Controller, I want to see an "Expected Exhaustion Date" for my budget (e.g., "Food budget will run out on the 22nd"), so that I can subconsciously slow down my spending.
- **Acceptance Criteria**:
  - Calculation: (Current Spending / Days into Month) \* Total Days in Month.
  - Burn rate should be visible as an "Info" badge on each category card.

---

## 4. Re-allocation & Flexibility

### US-BUD-005: "Inter-Category Swipe" Transfers

- **Story**: As a Budget Controller, I want to move 1M VND from my "Shopping" budget to my "Dining" budget because I overspent on dining, so that my _Total_ monthly spending limit remains unchanged.
- **Acceptance Criteria**:
  - Interface must allow "Swapping" budget amounts between categories.
  - Must ensure the total monthly allocation remains equal before and after the swap.

---

## 5. History & Learning

### US-BUD-006: Historical Variance Analysis

- **Story**: As a Budget Controller, I want to see a chart comparing my "Budget" vs. "Actual" spending over the last 6 months, so that I can identify categories where I am setting unrealistic targets.
- **Acceptance Criteria**:
  - Grouped Bar Chart (Budget vs. Actual).
  - Ability to filter by specific category (e.g., "Dining Out Trends").

---

## 6. AI-Assisted Strategic Planning

### US-BUD-007: AI-Powered Budget Suggestions & One-Click Approval

- **Story**: As a Budget Controller, I want the system to proactively suggest budget updates based on either (A) my actual 3-month spending history or (B) chosen financial frameworks (e.g., 5 Jars), so that I can keep my limits realistic and approve changes with one click.
- **Acceptance Criteria**:
  - AI must compute suggested amounts based on average income and the selected strategy (Historical/Framework).
  - UI must show a "Draft Proposal" side-by-side with current targets.
  - MUST include an **"Approve as Suggested"** button that batch-updates all category targets in Google Sheets immediately.
  - Must provide a "Partial Apply" option to only update specific categories.

### US-BUD-008: Interactive Budget Tuning Slider

- **Story**: As a Budget Controller, I want to use a slider to manually fine-tune each category's budget amount (e.g., sliding "Necessities" from 10M to 9M), so that I can intuitively see the "re-balancing" effect and the impact on my Net Savings Goal.
- **Acceptance Criteria**:
  - Each budget category row must include a horizontal slider (Min: 0, Max: [Monthly Income]).
  - Adjusting a slider must reflect the new VND amount in real-time.
  - The "Total Budgeted" and "Savings Potential" indicators must update instantly as the slider moves.
  - Changes must require a "Save Changes" confirmation to persist to Google Sheets.
