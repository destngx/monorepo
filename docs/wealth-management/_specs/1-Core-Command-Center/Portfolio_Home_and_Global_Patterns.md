# Specification: Portfolio Home & Command Center

**Author**: Product Owner / UX Strategist  
**Intended Audience**: Frontend Engineering / Design Team  
**Status**: Approved for Development  
**Keywords**: Situational Awareness, Executive Summary, Glassmorphism, Stealth Mode

---

## 1. Product Intent

To provide an immediate, high-fidelity synthesis of the user's total net worth, monthly financial velocity (Cash Flow), and prioritized AI-driven alerts. This is the **Command Center** for the entire financial ecosystem.

---

## 2. Global UI Patterns (Stealth & Privacy)

**PO Requirement**: The user's privacy is non-negotiable. The app must be "Safe for Public Use" by default.

### 2.1 Stealth Mode Implementation

- **Requirement**: All numeric balances and account identifiers must be masked (`••••••`) on initial load.
- **Trigger**: A "Privacy Toggle" (Eye Icon) in the Header reveals/hides data.
- **Acceptance Criteria (AC)**:
  - Masking must be applied to all child components using a global state (e.g., Zustand/Context).
  - State MUST reset to "Masked" upon client-side route changes or page refresh.
  - No data is persisted for this toggle (Security First).

---

## 3. Dashboard Functional Components

### 3.1 Net Worth Executive Summary

- **Visual**: Large-font total balance (VND/USD equivalent).
- **KPI**: 30-day delta (percentage and absolute growth).
- **AC**: Must render the aggregate total in < 300ms using cached Redis data.

### 3.2 Monthly Velocity (Cash Flow)

- **Visual**: Income vs. Expense comparison.
- **Logic**: Real-time sum of the `Transactions_YYYY` sharded ledger for the current month.
- **AC**: Must clearly calculate "Investment Fuel" (Net Profit for the Month).

### 3.3 Intelligence Greeting (AI Briefing)

- **Visual**: Narrative text block at the top.
- **Content**: Greet the user and provide 3 actionable bullet points (e.g., "Market is green," "3 Budget Items near limit").

---

## 4. Technical Constraints for Engineering

- **Glassmorphism**: All containers must use semi-transparent backgrounds with backdrop-blur.
- **Performance**: FCP (First Contentful Paint) < 1.2s.
- **Interactions**: Sub-300ms hover and click response transitions.

---

## 5. Success Metrics

- **Situational Awareness**: User can determine their total net worth within 2 seconds of page load.
- **Security**: 0% accidental exposure of private numbers in public settings.
- **Usage**: High frequency of the "Quick-Action Hub" triggers (Log, Analyze) from the home screen.
