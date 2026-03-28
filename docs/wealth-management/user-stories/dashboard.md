# User Stories: Dashboard Home (Mission Control)

## 1. Persona Definition: The Wealth Manager

- **Role**: Individual managing personal/family wealth with a professional, institutional mindset.
- **Needs**: High-level synthesis, instant situational awareness, actionable alerts, and privacy.

---

## 2. Net Worth Synthesis

### US-HOME-001: Net Worth Snapshot

- **Story**: As a Wealth Manager, I want to see my total net worth and its 30-day percentage change in the primary header, so that I can immediately understand the stability and growth of my capital at a single glance.
- **Acceptance Criteria**:
  - Net Worth must be calculated in VND.
  - Percentage change from 30 days ago should be displayed with color coding (Green for growth, Red for decline).
  - Accuracy of calculation must be within 0.1% based on all connected accounts.

### US-HOME-001a: Privacy Masking

- **Story**: As a Wealth Manager, I want to be able to hide the exact numerical values of my net worth with a "Eye" toggle, so that I can view the dashboard in public spaces without exposing my financial level to others.
- **Acceptance Criteria**:
  - Masking text must use standard `***` or blurred representation.
  - Toggled state should persist throughout the session for all currency-related components.

---

## 3. Financial Velocity (Cash Flow)

### US-HOME-002: Monthly Cash Flow Velocity

- **Story**: As a Wealth Manager, I want to see a "Monthly Velocity" indicator (Inflow vs. Outflow) for the current month, so that I can know how much "surplus capital" is remaining for investment or debt repayment.
- **Acceptance Criteria**:
  - Inflow must pull from all Income-tagged transactions.
  - Outflow must pull from all Expense-tagged transactions.
  - Calculation must update within 5 seconds of any transaction change.

---

## 4. Intelligent Context

### US-HOME-003: Daily AI Briefing Summary

- **Story**: As a Wealth Manager, I want a 3-bullet narrative summary at the top of my dashboard, so that I can understand the fundamental reason behind my wealth status changes without manually auditing multiple reports.
- **Acceptance Criteria**:
  - Summary should cover Portfolio Health, Market Sentiment, and Budget Alerts.
  - Must be generated using the latest data from the Intelligence API.

---

## 5. Navigation & Action

### US-HOME-004: Quick Action Hub

- **Story**: As a Wealth Manager, I want shortcuts for "Add Transaction" and "Stock Analysis" (Market Pulse) at the top of the home page, so that I can quickly record data or research ideas with minimal clicks.
- **Acceptance Criteria**:
  - Shortcuts MUST be persistent across all dashboard views if possible, or easily accessible from Home.
  - Triggers should open relevant modals or navigate to specialized terminals.
