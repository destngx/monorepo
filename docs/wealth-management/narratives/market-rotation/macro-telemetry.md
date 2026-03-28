# Details: Macro Telemetry - The Early Warning Spark

## 1. Feature Meaning: The Macro Pulse

This feature represents the **"Ear-to-the-Ground"** for the Vietnamese macro-economy.

## 2. API Source & Logic

- **Primary Source**: `GET /api/market-pulse` (action: `VNINDEX`, `SJC Gold`).
- **Initial Signal**: High-volume selling in the stock market (captured by `VNINDEX` ticker) accompanied by a breakout in **SJC Gold (SJC 9999)** prices via `Vang.Today`.

## 3. Business Use Cases (Actionable)

- **Early Warning System**: Alerts the user when "Macro Anxiety" is rising in the market.
- **Flight Detection**: Automatically triggers a `News Analyze` query for labels like "Inflation Vietnam" or "Interest Rate Hike."

## 4. Why This Hub?

The **Market Pulse Terminal** is the entry point for daily monitoring; it acts as the "Spark" that triggers the rest of the **Rotation Signal** pipeline.
