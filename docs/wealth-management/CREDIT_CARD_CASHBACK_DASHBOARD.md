# Credit Card Cashback Optimization Dashboard

## Overview

The **Credit Card Usage Dashboard** is a comprehensive financial optimization tool designed specifically for managing Sacombank's dual credit card ecosystem. It helps you maximize cashback earnings by providing real-time insights into spending patterns, cashback rules, and optimization strategies.

---

## Feature Highlights

### ✅ Dual-Card Management

- **Sacombank Visa Platinum Cashback** — Optimized for online & overseas spending
- **Sacombank Visa UNIQ Platinum** — Optimized for supermarkets & transport

### ✅ Intelligent Cashback Calculation

- Automatic categorization of transactions by type
- Real-time cashback estimation with official Sacombank rules
- Cap tracking (600K VND per cycle, 7.2M VND per year, per card)

### ✅ Usage Analytics

- Current month spending overview
- Monthly history (utilization trends & cashback earned)
- Top spending categories
- Credit utilization percentage

### ✅ Optimization Tips

- Smart alerts for low/high utilization
- Recommended routing strategy
- Cashback efficiency guide

---

## Official Sacombank Cashback Rules

### 1. Sacombank Visa Platinum Cashback

| Category                                | Cashback Rate | Cap/Cycle | Cap/Year |
| --------------------------------------- | ------------- | --------- | -------- |
| **Online** (Shopee, Lazada, Tiki, etc.) | **5%**        | 600K VND  | 7.2M VND |
| **Overseas** (International POS)        | **3%**        | Shared    | Shared   |
| **Airline** (+ all other spending)      | **0.5%**      | Shared    | Shared   |

**Key Conditions:**

- Card must be active on cashback credit date
- Consumer purchases only (no cash advances, fund transfers, or insurance payments)
- Cashback credited: Day 10–15 of next month
- Minimum: 10K VND/cycle (if earned) · Maximum: 600K VND/cycle
- Refunded transactions clawed back

[📄 Official Document](https://www.sacombank.com.vn/content/dam/sacombank/files/ca-nhan/the/tin-dung/Quy-dinh-hoan-tien-Visa-PlatinumCashback.pdf)

---

### 2. Sacombank Visa UNIQ Platinum

| Category                                  | Cashback Rate | Cap/Cycle               | Cap/Year |
| ----------------------------------------- | ------------- | ----------------------- | -------- |
| **Supermarket** (CoopMart, WinMart, BigC) | **20%**       | 600K VND                | 7.2M VND |
| **Transport** (Grab, taxi, fuel)          | **20%**       | Shared with Supermarket | Shared   |
| **Other Spending**                        | **0.5%**      | Shared                  | Shared   |

**Special Rules:**

- **First 30 days of activation**: Supermarket + Transport earn **50%** (not 20%)
- Supermarket & Transport share the same 600K cap (combined)
- After 30 days: Both drop to 20%

**Key Conditions:**

- Card must be active on cashback credit date
- Consumer purchases only
- Only qualifying merchants (by MCC) earn cashback
- Cashback credited: Day 10–15 of next month
- Minimum: 10K VND/cycle · Maximum: 600K VND/cycle
- Refunded transactions clawed back

[📄 Official Document](https://www.sacombank.com.vn/content/dam/sacombank/files/ca-nhan/the/tich-hop/DKDK%20HOAN%20TIEN%20THE%20SACOMBANK%20VISA%20UNIQ%20PLATINUM%20267.pdf)

---

## Dashboard Sections

### 1. Summary Cards (Top Row)

Four quick-stat cards showing:

- **Total Limit**: 40M VND (combined)
- **Used This Month**: Current spending + utilization %
- **Est. Cashback**: Total estimated cashback earned (both cards)
- **Remaining**: Available credit across both cards

---

### 2. Individual Card Details

Each card gets its own detailed panel showing:

#### Card Header

- Card name & icon
- **Usage percentage** (prominently displayed)

#### Usage Bar

- Visual progress bar (green <50%, orange 50-80%, red >80%)
- Current spending amount
- Shows which card is being utilized more

#### Limit & Available

- Credit limit (20M each)
- Available credit remaining

#### Estimated Cashback Box

- Green highlight box showing total estimated cashback for current month
- Updated in real-time as transactions are processed

#### Top 5 Spending Categories

- Category name
- Amount spent in category
- Estimated cashback earned in that category
- **Example:**
  ```
  Online Shopping       ₫1,500,000
                        +₫75,000 (5% cashback)
  ```

#### Monthly Efficiency Table

- Shows last 3 months of:
  - Month (e.g., "Feb '26")
  - Utilization % for that month
  - Cashback earned that month
- Identifies trending patterns

#### Optimization Alerts

- **Low Usage (<30%)**: "Consider using this card more for cashback optimization"
- **High Usage (>80%)**: "Be cautious of credit limit"

#### Transaction Count

- Number of transactions this month

---

### 3. Optimization Guide (Bottom Card)

Two-column guide showing:

**Sacombank Visa Platinum Cashback**

```
Online                  5% cashback
Overseas                3% cashback
Airline (+ other)       0.5% cashback
```

**Sacombank Visa UNIQ Platinum**

```
Supermarket             20% cashback
Transport               20% cashback
Other Spending          0.5% cashback
```

---

## Optimal Spending Strategy

### Route to UNIQ Platinum (20% cashback)

- Supermarkets: CoopMart, WinMart, BigC
- Transport: Grab, taxi, fuel stations, parking
- **Why**: 20% is the highest rate available; maximize this first

### Route to Platinum Cashback (5% cashback)

- Online shopping: Shopee, Lazada, Tiki
- Food delivery: Grab Food, Baemin, Now
- Subscriptions: Netflix, Spotify, Coursera
- International purchases: Always use this card for overseas transactions
- **Why**: 5% online is 25x better than 0.5% fallback

### Flexible Routing (when one card is near cap)

- Either card can handle "Other Spending" at 0.5% when either reaches its 600K cap
- Check the dashboard to see which card has remaining cap headroom

---

## Earning Potential

With optimal usage, you can earn:

| Scenario                 | Monthly         | Annually     |
| ------------------------ | --------------- | ------------ |
| **One card maxed**       | 600K VND        | 7.2M VND     |
| **Both cards maxed**     | 1,200K VND      | 14.4M VND    |
| **Realistic dual usage** | 800K–1,000K VND | 9.6M–12M VND |

**Realistic Scenario** (Your actual data):

- 10M VND/month spending
- Split 60/40 (online vs supermarket/transport)
- Both cards heavily used
- **Expected: 750K–900K VND/month cashback**

---

## How the Dashboard Calculates Cashback

### 1. Transaction Categorization

The component automatically maps your transaction categories to cashback rules:

**For Platinum Cashback:**

- Keywords like "shopee", "lazada", "tiki", "delivery", "netflix" → **Online (5%)**
- Keywords like "airline", "flight" → **Airline (0.5%)**
- Everything else → **Other (0.5%)**

**For UNIQ Platinum:**

- Keywords like "coop", "winmart", "bigc", "supermarket" → **Supermarket (20%)**
- Keywords like "grab", "taxi", "fuel", "transit", "gas" → **Transport (20%)**
- Everything else → **Other (0.5%)**

### 2. Cap Management

- Cashback is capped at **600K VND per cycle** (billing period)
- If a category would earn > cap, it's reduced to exactly 600K
- Different cards have separate caps (not combined)
- Annual cap: **7.2M VND** (12 months × 600K)

### 3. Efficiency Calculation

Monthly efficiency shows:

- **Utilization %**: (Total spent / 20M limit) × 100
- **Cashback earned**: Sum of all category cashbacks for the month

---

## Key Insights from Your Data

### What the Dashboard Shows You

1. **Which card is being used more** → Utilization % in header
2. **How much of each cashback cap is being used** → "Est. Cashback" box
3. **Best spending categories for each card** → "Top Categories" section
4. **Monthly trends** → "Monthly Efficiency" showing if you're spending more/less
5. **How close you are to caps** → If a card shows 600K+ cashback, you've likely maxed the cap

### Example Interpretation

If you see:

- **UNIQ Platinum**: 95% utilization, 600K estimated cashback
  - ✅ You've maxed out the supermarket/transport cap
  - 📊 Consider shifting non-qualifying spending to Platinum Cashback

- **Platinum Cashback**: 30% utilization, 50K estimated cashback
  - ⚠️ You're underutilizing online spending potential
  - 💡 Route more online purchases here (Shopee, Lazada, Netflix, food delivery)

---

## Limitations & Notes

### What's NOT Included

1. **MCC (Merchant Category Code) Validation**
   - Some merchants may not qualify even if category matches
   - Sacombank categorizes by MCC, not merchant name
   - Dashboard uses keywords as best approximation

2. **Overseas Transaction Detection**
   - Dashboard can't automatically detect overseas transactions
   - Manually tag with "Overseas" or "International" for accurate 5% calculation
   - POS swipes abroad get 3%; online international gets 5% (both Platinum Cashback)

3. **First 30-Day Bonus (UNIQ Platinum)**
   - Dashboard shows 20% for supermarket/transport
   - You actually earned 50% in first 30 days of card activation
   - Historical data will reflect 50% if card is new

4. **Refund Clawback**
   - If a transaction is refunded, cashback is clawed back automatically
   - Dashboard shows original earned amount; actual payout may be lower
   - Check your next statement for clawbacks

### Data Requirements

For accurate cashback calculation, your transactions should include:

- **Account Name**: "Credit Card Sacombank" or "Credit Card Sacombank UNIQ"
- **Category**: Spend category (e.g., "Lunch", "Shopee", "Grab")
- **Amount**: Transaction amount in VND
- **Date**: Transaction date (for monthly grouping)

---

## Using the Dashboard

### Daily Check-In

- See how much you've spent and earned in cashback today
- Monitor approach to 600K monthly cap

### Monthly Review

- View all 3 months of spending/cashback history
- Identify which card is earning more
- Spot categories where you're earning high cashback

### Optimization Decisions

- **Before major purchases**: Check which card has cap room remaining
- **When choosing payment method**: Refer to the Optimization Guide
- **When refunding**: Be aware cashback will be clawed back

### Data Export

- Numbers displayed are for reference only
- For official Sacombank records, check your statement

---

## FAQ

**Q: Why does my actual cashback differ from the dashboard estimate?**
A: Possible reasons:

- MCC rejection (merchant didn't qualify despite category match)
- Refund clawback from previous month
- Transaction settled later than displayed date
- Minimum 10K VND threshold not met

**Q: How often is cashback credited?**
A: Day 10–15 of the following month. The dashboard shows estimated amounts; actual credit depends on Sacombank processing.

**Q: Can I earn cashback on the same purchase with both cards?**
A: No. You use one card per transaction. The dashboard helps you choose the optimal card for maximum cashback.

**Q: What happens if I max both caps in one month?**
A: You've earned the maximum possible (1,200K VND). Further spending earns no additional cashback until the next billing cycle.

**Q: How does the shared cap work on UNIQ (Supermarket + Transport)?**
A: Together they share 600K per cycle. If you spend 300K on supermarket, you only have 300K cap room left for transport. The dashboard shows combined progress.

**Q: Why isn't my online purchase showing as "Online" category?**
A: The dashboard uses transaction category tags. If you tagged it differently (e.g., "Food / Goods" instead of "Online"), it won't map correctly. Retag for accurate calculation.

---

## Support & Updates

- **Cashback rules updated**: Sacombank occasionally changes rates. Check the official PDF links above.
- **Dashboard accuracy**: Depends on accurate transaction categorization in your data.
- **Feature requests**: Consider tagging additional merchants or creating custom categories for better accuracy.

---

## Summary

The **Credit Card Cashback Optimization Dashboard** transforms raw spending data into actionable financial intelligence. By understanding your two cards' complementary strengths (UNIQ for supermarkets/transport, Platinum for online/overseas), you can systematically optimize your spending to earn the maximum possible cashback while staying within safe credit utilization limits.

**Target**: 800K–1,000K VND/month in cashback with disciplined, two-card routing.
