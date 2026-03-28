# Details: Budget Velocity Correlation

## 1. Feature Meaning: The Flow of Surplus

The **Budget Velocity** represents how well the user is "Scaling" their wealth relative to their daily "Expenses."

## 2. API Source & Logic

- **Primary Source**: `GET /api/budget` (action: `monthlyRemaining`, `categoryType`).
- **Velocity Signal**: The "Gap" between **Budget Remaining** and **Transaction Deposit** represents the surplus available to flow into "Brokerage Savings."

## 3. Business Use Cases (Actionable)

- **Spend Heatmaps**: Visualizes **Budget versus Spending** targets as a percentage of overall net worth.
- **AI Savings Recommendation**: The `Intelligence Briefing` uses this budget surplus to recommand shifting money into **Fmarket Bond Funds**.

## 4. Why This Hub?

**The Budget API** represents the "Future Investment Potential" of the user. It is the fuel for the rest of the **Portfolio Valuation** cycle.
