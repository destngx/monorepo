# Details: Real-time Asset Pricing & Mark-to-Market

## 1. Feature Meaning: The Market Reality

**Real-time Asset Pricing** transforms a "Static Stock Number" into a "Live Asset Value."

## 2. API Source & Logic

- **Primary Source**: `POST /api/ticker/analyze` (action: `Ticker Price`, `Technicals`).
- **Valuation Logic**: Multiplies the `Balance` from `Accounts` sheet by the `Live Price` from the `Ticker` API.

## 3. Business Use Cases (Actionable)

- **Net Worth Accuracy**: Dynamically updating the **Portfolio Value** in the header so it reflects current market prices for VCB, FPT, etc.
- **Wyckoff Positioning**: Visualizing the "Mark-to-Market" value in context of the ticker's **"Market Phase"** (Markup/Accumulation).

## 4. Why This Hub?

This is the "Real-time Update" of the static **Portfolio Valuation**. It ensures that "Yesterday's Cost" doesn't mislead "Today's Net Worth."
