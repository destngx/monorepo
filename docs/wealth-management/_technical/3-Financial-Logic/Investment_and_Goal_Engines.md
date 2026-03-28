# Specification: Financial Logic & Advanced Algorithms

**Author**: Product Owner / Wealth Management BA  
**Intended Audience**: Engineering Team (Backend & Logic)  
**Status**: Mid-Stage Implementation  
**Keywords**: Goal Projections, Investment Math, Rebalancing, Compounding

---

## 1. Business Objective

To automate the complex mathematical calculations involved in wealth planning, transforming absolute numbers (Net Worth) into directional time-based targets (Financial Freedom).

---

## 2. Wealth Trajectory & Goal Engine

**BA Requirement**: The system must answer the fundamental question: "When will I be financially free?"

### 2.1 The "Freedom Score" Algorithm

- **Formula**: `(Net Worth / Monthly Expenses) = Freedom Months`.
- **Projection Logic**:
  - `Future Value (FV) = PV * (1 + r)^n`.
  - **r** (Real Return): (Nominal Return - Inflation (6% VND / 2% USD)).
  - **n** (Timeframe): Monthly granularity.

### 2.2 Strategic Saving Simulation

- **Logic**: User can input "Monthly Contribution" to simulate how fast the "Freedom Goal" is reached.
- **PO Constraint**: The system MUST account for the **S-Curve** of compounding, showing accelerated growth after year 5.

---

## 3. Investment Terminal (Market Alpha) Logic

**BA Requirement**: The investment module must combine **Market Pulse** with **Personal Portfolio State**.

### 3.1 Asset Allocation Drift (Rebalancing)

- **Desired Target**: Defined per account category (e.g., 50% Stocks, 20% Crypto, 30% Cash).
- **Drift Detection**: Triggered when any class deviates by > 5% from its target.
- **Actionable Advice**: "Sell X Amount of over-performing asset; Buy Y Amount of under-performing asset."

### 3.2 Valuation Models (Stock Research)

The dev team is responsible for implementing the following standard models:

1.  **DDM (Dividend Discount Model)**: For high-yield tickers (e.g., banks in VNStock).
2.  **P/E Relative Valuation**: Comparing current P/E to the 3-year historical average for the sector.

---

## 4. Normalization & Currency Conversion

**PO Requirement**: Total portfolio value must be normalized into the **Base Currency (VND)** using live spot rates.

- **USD/VND**: Use Commercial Bank Sell rates.
- **Crypto/VND**: Use Binance/P2P live spot rates (USDT/VND).
- **Alpha Stock**: Use HOSE/HNX live prices (VNStock API).

---

## 5. Engineering Success Metrics

- **Math Accuracy**: Zero variance between spreadsheet calculations and UI projections.
- **Performance**: Goal re-calc must happen in < 500ms when user sliders are moved.
- **Robustness**: Graceful fallback to "Historical Averages" (e.g., 8% Stock Market return) if real-time projections fail.
