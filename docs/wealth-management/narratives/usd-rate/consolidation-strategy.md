# Details: USD Rate Consolidation Strategy

## 1. Feature Meaning

Our app currently makes multiple calls to external USD APIs. **Consolidation** means moving to a single **"Source of Truth"** layer for all exchange rate needs.

## 2. Technical Recommendation: "The Fmarket Core"

- **Primary Source**: Standardize all components to use the `POST /api/fmarket` (action: `getUsdRateHistory`) endpoint.
- **Redundancy Phaseout**: Remove redundant calls to other third-party exchange rate APIs, reducing external dependency and enhancing performance.

## 3. Integrated Architecture

The `market-data-service` in the Shared Lib should be updated to a **Unified Rate Service**:

- **Official Interface**: `getOfficialUsdRate()` -> Calls Fmarket.
- **Market Interface**: `getMarketUsdRate()` -> Calls Yahoo/Finance/Vang.Today and potentially Binance in the future.

## 4. Consolidating the Use Cases

- **Investment Insight**: Uses the "Unified Rate" to value offshore ETFs.
- **Fmarket Insight**: Already uses this API for mutual fund NAV history.
- **Accounts**: Should be updated to dynamically fetch this rate instead of relying on manual entry or sheet formulas.
