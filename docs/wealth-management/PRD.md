# Product Requirements Document (PRD)

## Wealth Management System — Personal Finance Platform

**Version:** 1.0
**Last Updated:** 2026-02-28
**Status:** Planning

---

## 1. Overview

A personal wealth management web application that connects to your existing Google Sheets financial data, provides intelligent dashboards, and uses AI to automate categorization, analysis, and financial advice. The system tracks both spending (daily budgets) and wealth building (investments, net worth, assets) across Vietnamese bank accounts (VND), cryptocurrency holdings, gold, savings deposits, and other investment vehicles. Designed for a single user managing personal finances with AI-powered insights and projections.

### 1.1 Problem Statement

- Manual transaction tracking in Google Sheets is tedious and error-prone
- Budget categories are poorly defined, making spending analysis unreliable
- No automated categorization — every transaction requires manual input
- No unified net worth view across VND bank accounts and USDT crypto holdings
- No intelligent insights or alerts about spending patterns
- Bank email notifications contain transaction data but aren't utilized

### 1.2 Solution

A web-based system that:

- Syncs bidirectionally with your existing Google Sheets (preserving your workflow)
- Provides rich dashboards for net worth, budgets, and spending analysis
- Uses AI to auto-categorize transactions, analyze spending, and provide financial advice
- Supports flexible AI provider switching (OpenAI, Google Gemini, Anthropic)
- Will eventually auto-parse bank email notifications into transactions

---

## 2. User Personas

### Primary User: Personal Finance Manager (You)

- **Context:** Vietnamese professional managing personal finances and building long-term wealth
- **Accounts:** Vietnamese bank accounts (VND), Binance crypto (individual coins), physical gold (SJC), savings deposits, potential stocks, real estate, business equity
- **Current workflow:** Google Sheets for tracking, manual entry
- **Pain points:** Manual categorization, no insights, no automation
- **Goals:** Automated tracking, smart budgeting, wealth portfolio management, investment growth, achieving financial independence, AI-powered financial advisor

---

## 3. Features

### 3.1 Phase 1 — Core Platform (MVP)

#### F1: Dashboard

- **Net worth overview:** Total across all accounts, converted to VND (with live USDT/VND rate)
- **Account balances:** Each account with type, balance, currency
- **Spending summary:** Current month vs. budget, top categories
- **Trends:** Monthly spending chart (last 6–12 months)
- **Quick stats:** Income vs. expenses, savings rate, biggest expense category

#### F2: Account Management

- **View all accounts** synced from Google Sheets
- **Account types:** Bank (VND), Crypto (USDT), Cash, Investment
- **Balance tracking** over time with historical chart
- **Goal tracking** (savings goals with progress percentage)
- **Multi-currency display** with VND as base currency

#### F3: Transaction Management

- **List view** with filtering (by date range, account, category, amount)
- **Add/edit transactions** (syncs back to Google Sheets)
- **AI auto-categorization** on new transactions
- **Bulk re-categorization** with AI suggestions
- **Search** by payee, memo, category
- **Payment vs. Deposit** tracking with running balance

#### F4: Budget Management

- **Monthly budgets** by category
- **Yearly budgets** by category
- **Budget vs. Actual** comparison with visual indicators
- **AI-suggested budget adjustments** based on spending history
- **Overspending alerts** (visual, in-app)
- **Category management** (add, edit, merge categories)

#### F5: AI Chat Interface

- **Natural language Q&A** about your finances
  - "How much did I spend on food this month?"
  - "What's my savings rate?"
  - "Compare my spending this month vs. last month"
  - "Am I on track for my yearly budget?"
- **Financial advice** based on your actual data
  - "How can I reduce spending?"
  - "Should I adjust my food budget?"
- **Transaction analysis**
  - "Show me my largest expenses this month"
  - "What are my recurring payments?"
- **Provider switching:** Change between OpenAI / Gemini / Anthropic from settings

#### F6: Google Sheets Sync

- **Read** account balances, budgets, and transactions from existing sheets
- **Write** new/edited transactions back to the sheet
- **Bidirectional sync** — sheet remains source of truth
- **Conflict handling** — detect and resolve sync conflicts
- **Auto-refresh** on dashboard load + manual refresh button

#### F7: Spending Categories (AI-Enhanced)

A well-structured category taxonomy (see Section 5) that:

- Replaces poorly defined existing categories
- Supports AI auto-assignment
- Allows custom subcategories
- Maps from your existing sheet categories to new ones

#### F12: Investment Portfolio Tracker

- **Manual asset entry:** Track crypto (individual coins beyond USDT total), stocks, gold (physical/SJC), savings deposits (kỳ hạn), real estate, business equity
- **Per-asset details:** Name, type, quantity, purchase price (cost basis), current value, currency
- **Portfolio overview:** Total portfolio value in VND, asset-by-asset breakdown with current vs. purchase value
- **P&L per asset:** Unrealized gain/loss (current vs. cost basis), realized gains when sold
- **Binance integration:** Auto-sync individual coins (BTC, ETH, etc.) from holdings via API
- **Live price updates:** CoinGecko for crypto, free sources for gold and stocks

#### F13: Asset Allocation & Net Worth Composition

- **Asset class breakdown:** Pie/donut chart showing % in cash, crypto, gold, savings deposits, stocks, real estate, other
- **Net worth composition:** Visual breakdown of where wealth sits across asset classes
- **Allocation targets:** Set target allocation (e.g., 30% crypto, 40% cash, 20% gold, 10% other) and view deviation from targets
- **Rebalancing suggestions:** AI recommends asset moves to hit target allocation

#### F14: Net Worth History & Projection

- **Historical snapshots:** Monthly net worth tracking showing total net worth over time
- **Growth trends:** Month-over-month and year-over-year net worth change with growth rate
- **AI projections:** Based on savings rate, investment returns, spending patterns, project net worth at 1, 3, 5, 10 years
- **What-if scenarios:** Run scenarios like "If I save extra 5M VND/month" or "If crypto drops 30%" to see impact

#### F15: Financial Goals

- **Goal creation:** Name, target amount, target date, linked asset/account
- **Goal types:** Emergency fund, retirement, home purchase, vacation, education, custom
- **Progress tracking:** Visual progress bar showing current amount vs. target
- **AI-powered planning:** Calculates required savings rate and investment returns to reach goals
- **Priority ranking:** AI-ranked goals by urgency and impact

#### F16: Financial Health Score

- **Composite score (0-100):** Calculated from savings rate, emergency fund coverage, debt-to-income ratio, investment diversification, budget adherence, net worth growth rate
- **AI action items:** Specific recommendations like "Your diversification is low (85% crypto). Consider allocating to gold or savings deposits."
- **Health score history:** Track score over time to measure financial progress

#### F17: Passive Income Dashboard

- **Passive income streams:** Track staking rewards, interest from savings deposits, dividends, rental income
- **Monthly passive income total:** Sum of all passive income across accounts
- **Financial independence ratio:** Passive income vs. monthly expenses (progress toward FI)
- **Trend chart:** Visualize growing passive income over time

#### F18: Wealth AI Advisor (Enhanced AI Chat)

- **All F5 capabilities PLUS:**
- **Portfolio analysis:** "What's my portfolio return this year?" "Which asset is performing best?"
- **Goal tracking:** "Am I on track for my emergency fund?" "When will I hit my 1B VND goal?"
- **Net worth projection:** "Project my net worth in 5 years" "What if I increase savings by 5M/month?"
- **Rebalancing advice:** "How should I rebalance my portfolio?" "My allocation is off target"
- **Financial health insights:** "What should I improve in my finances?" "My emergency fund is low"
- **What-if analysis:** "What if I invest extra 3M VND/month in crypto?" "Impact of market downturn?"

### 3.2 Phase 2 — Automation (Future)

#### F8: Bank Email Parsing

- Connect to email (Gmail API or IMAP)
- Parse bank notification emails (VietcomBank, etc.)
- Auto-extract: date, amount, payee, account
- Auto-create draft transactions for review
- AI categorization on parsed transactions

#### F9: Proactive AI Alerts

- Weekly spending summary pushed to dashboard
- Overspending warnings (approaching budget limit)
- Unusual transaction detection (amount outliers)
- Recurring payment reminders
- Bill due date reminders (from Accounts "Date to pay" field)

#### F10: Advanced Analytics

- Spending forecasting (ML-based)
- Category trend analysis
- Income vs. expense projections
- Crypto portfolio tracking (beyond just balance)
- Net worth projection

#### F11: Recurring Transaction Detection

- AI identifies recurring patterns (rent, subscriptions, salary)
- Auto-suggest recurring transaction creation
- Alert on missed recurring transactions

---

## 4. User Stories

### Dashboard

- **US-1:** As a user, I want to see my total net worth in VND so I understand my financial position at a glance.
- **US-2:** As a user, I want to see each account's balance so I know where my money is.
- **US-3:** As a user, I want to see this month's spending vs. budget so I know if I'm on track.
- **US-4:** As a user, I want to see spending trends over the last 6 months so I can spot patterns.

### Transactions

- **US-5:** As a user, I want new transactions auto-categorized so I don't have to manually assign categories.
- **US-6:** As a user, I want to filter transactions by date, category, and account for analysis.
- **US-7:** As a user, I want to add a transaction in the app and have it sync to Google Sheets.
- **US-8:** As a user, I want to bulk re-categorize old transactions using AI suggestions.

### Budget

- **US-9:** As a user, I want to see monthly budget progress per category with visual bars.
- **US-10:** As a user, I want AI to suggest budget adjustments based on my actual spending.
- **US-11:** As a user, I want to set both monthly and yearly budget limits per category.

### AI Chat

- **US-12:** As a user, I want to ask natural language questions about my finances and get accurate answers.
- **US-13:** As a user, I want to switch AI providers (OpenAI, Gemini, Anthropic) from settings.
- **US-14:** As a user, I want the AI to reference my actual data when giving advice.

### Google Sheets

- **US-15:** As a user, I want changes in Google Sheets to reflect in the app automatically.
- **US-16:** As a user, I want changes in the app to sync back to Google Sheets.
- **US-17:** As a user, I want my Google Sheet to remain the source of truth.

### Wealth Management

- **US-18:** As a user, I want to track all my assets (crypto, gold, savings, stocks, real estate) in one place so I can see my total wealth.
- **US-19:** As a user, I want to see my asset allocation as a pie chart so I understand my diversification and risk profile.
- **US-20:** As a user, I want to set financial goals and track progress toward them so I can plan for the future.
- **US-21:** As a user, I want AI to project my net worth growth based on my current savings and investment habits.
- **US-22:** As a user, I want a financial health score that tells me what to improve in my financial situation.
- **US-23:** As a user, I want to track my passive income streams and see progress toward financial independence.
- **US-24:** As a user, I want to ask the AI wealth advisor complex questions like "How should I rebalance?" and get portfolio-specific advice.
- **US-25:** As a user, I want to run "what-if" scenarios to see how changes to savings or investments affect my future net worth.
- **US-26:** As a user, I want to track my monthly debt repayments and see my progress toward clearing loans.

---

## 5. Spending Categories

### Category Taxonomy

The following replaces the current poorly-defined categories in your Google Sheet. Each top-level category has common subcategories. AI will auto-assign transactions to the most specific match.

| #   | Category                  | Subcategories                                                                           | Examples                                           |
| --- | ------------------------- | --------------------------------------------------------------------------------------- | -------------------------------------------------- |
| 1   | **Housing**               | Rent, Utilities (Electric/Water/Gas), Internet, Maintenance, Home Insurance, Furnishing | Monthly rent, electric bill, plumber               |
| 2   | **Food & Dining**         | Groceries, Restaurants, Delivery (GrabFood/ShopeeFood), Coffee & Drinks, Snacks         | Bách Hóa Xanh, Highland Coffee, GrabFood           |
| 3   | **Transportation**        | Fuel/Petrol, Grab/Taxi, Public Transit, Parking, Vehicle Maintenance, Vehicle Insurance | Grab ride, petrol station, bike repair             |
| 4   | **Healthcare**            | Health Insurance, Doctor Visits, Pharmacy/Medicine, Dental, Vision, Mental Health       | Hospital visit, pharmacy purchase                  |
| 5   | **Personal Care**         | Clothing, Grooming/Haircut, Gym/Fitness, Personal Items                                 | Barber, gym membership, new clothes                |
| 6   | **Entertainment**         | Streaming (Netflix/Spotify/YouTube), Games, Hobbies, Social Events, Movies/Concerts     | Netflix subscription, concert ticket               |
| 7   | **Education**             | Online Courses, Books, Certifications, Workshops, School/University                     | Udemy course, book purchase                        |
| 8   | **Shopping**              | Electronics, Household Items, Gifts, Online Shopping                                    | Shopee order, birthday gift, new phone             |
| 9   | **Bills & Subscriptions** | Phone Plan, SaaS/Software, Memberships, Recurring Services                              | Viettel mobile, iCloud, ChatGPT Plus               |
| 10  | **Financial**             | Bank Fees, Crypto Exchange Fees, Interest Payments, Loan Payments, Wire Transfer Fees   | Binance trading fee, bank transfer fee             |
| 11  | **Travel**                | Flights, Hotels/Accommodation, Activities, Travel Food, Travel Transport                | Vietnam Airlines, Booking.com, travel Grab         |
| 12  | **Work & Business**       | Software/Tools, Coworking, Professional Services, Equipment, Business Meals             | GitHub subscription, coworking space               |
| 13  | **Savings & Investments** | Emergency Fund, Retirement, Crypto Purchase, Stock Purchase, Gold                       | Monthly BTC DCA, savings deposit                   |
| 14  | **Income**                | Salary, Freelance, Investment Returns, Interest, Crypto Gains, Cashback, Gifts Received | Monthly salary, freelance payment, staking rewards |
| 15  | **Transfers**             | Between Own Accounts, To/From Binance, Peer-to-Peer                                     | VCB → Binance, savings transfer                    |
| 16  | **Other**                 | Uncategorized, Miscellaneous                                                            | Anything AI can't confidently categorize           |

### Category Mapping

AI will handle mapping from your current Google Sheet categories to this new taxonomy. Existing categories will be preserved in the `Tag` field as a reference during migration.

### Category Rules for AI

1. **Confidence threshold:** AI assigns category if >80% confident, otherwise suggests top 3 options for manual selection
2. **Learning:** Category corrections feed back to improve future predictions
3. **Context-aware:** Same payee can have different categories (e.g., "Grab" → Transportation OR Food & Dining based on amount/memo)
4. **Transfer detection:** AI detects transfers between own accounts vs. actual expenses

---

## 6. Non-Functional Requirements

### Performance

- Dashboard loads in <2 seconds
- Google Sheets sync completes in <5 seconds
- AI responses stream in real-time (first token <1 second)

### Security

- Google OAuth 2.0 for Sheets access
- API keys stored server-side only (.env)
- No financial data stored in browser localStorage (except chat history)
- In-memory caching for performance (no persistent database needed)

### Usability

- Responsive design (desktop-first, mobile-friendly)
- Dark mode support
- Vietnamese Dong formatting (1,000,000 ₫)
- USDT formatting ($1,500.00)
- Date format: DD/MM/YYYY (Vietnamese convention)

### Reliability

- Google Sheets as source of truth (app data is derived)
- Graceful degradation when AI provider is unavailable
- Offline-capable dashboard (cached data)

---

## 7. Success Metrics

| Metric                           | Target                                     |
| -------------------------------- | ------------------------------------------ |
| Time to categorize a transaction | <1 second (AI) vs. ~30 seconds (manual)    |
| Budget tracking accuracy         | >95% of transactions correctly categorized |
| Dashboard load time              | <2 seconds                                 |
| AI response relevance            | >90% useful responses (subjective)         |
| Google Sheets sync reliability   | >99% successful syncs                      |

---

## 8. Constraints & Assumptions

### Constraints

- Single user system (no multi-tenant)
- Google Sheets is the data backbone (not a traditional database)
- AI costs are per-API-call (budget consideration for provider choice)
- Vietnamese banking emails have varied formats (parser must be flexible)

### Assumptions

- User has a Google Cloud project with Sheets API enabled
- User has API keys for at least one AI provider (OpenAI or Gemini)
- Google Sheet structure matches the documented format (Section in GOOGLE-SHEETS-INTEGRATION.md)
- USDT/VND exchange rate available via free API (e.g., CoinGecko)

---

## 9. Out of Scope (for Phase 1)

- Multi-user / family sharing
- Mobile native app (web responsive only)
- Direct bank API connections (Plaid/Tink — not available in Vietnam)
- Tax calculation or reporting
- Automated trading or trade execution
- Tax optimization strategies (consult a professional)
