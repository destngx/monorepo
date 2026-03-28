# Use Case Specification: Intelligence Center (AI Synthesis)

## UC-INT-001: Daily Strategic Briefing (Morning Login)

### 1. Summary

The user logs in at 9:00 AM and gets a synthesized view of their wealth, monthly budget status, and market context.

### 2. Actors

- **Primary Actor**: AI-First Investor (User)
- **Secondary Actor**: Intelligence API, Accounts API, Transactions API, Market Pulse API.

### 3. Preconditions

- System is authenticated and can fetch API data for all 3 areas (Portfolio, Budget, Market).

### 4. Basic Flow

1.  User enters the dashboard.
2.  System parallel-fetches "Total Net Worth," "Budget Progress," and "Market Status."
3.  System passes this data to the `Generative AI` engine as context.
4.  System displays "Good Morning! Here's your mission summary:"
5.  System renders 3-5 high-impact bullets:
    - "Your FPT holding is up +2.5% overnight. Net worth has grown."
    - "You have 3 million VND left in your 'Dining' budget for the month."
    - "Market Sentiment is 'Bullish' on rising VCB and HPG momentum."
    - "Action: 4 un-categorized Grab transactions found from yesterday."

### 5. Postconditions

- User has a complete situational awareness of their portfolio status without clicking multiple pages.

---

## UC-INT-002: Natural Language Expenditure Query (The Grab Search)

### 1. Summary

User asks the AI a specific question about their spending history instead of using filters.

### 2. Actors

- **Primary Actor**: AI-First Investor (User)
- **Supporting System**: Transactions API Tool.

### 3. Basic Flow

1.  User clicks the "Chat with Advisor" icon.
2.  User types: "How much did I spend at Grab in March 2024?"
3.  System identifies the "Intent" as a "Ledger Search."
4.  System triggers a `searchTransactions(keyword="Grab", dateRange="03/24")` Tool Call.
5.  System receives a list of 12 transactions totaling 540,000 VND.
6.  AI analyzes the Tool output and synthesizes the response.
7.  AI replies: "You had 12 Grab rides totaling 540,000 VND in March. This is 15% lower than your February spending."
8.  System renders an interactive "View Transactions" link inside the chat.

### 4. Alternative/Exception Flows

- **Ambiguous Query (E1)**: If user asks "Show me Grab," and there are transactions in multiple years, AI asks for clarification on the time period.
- **No Results (E2)**: If no matching transactions are found, AI helpfully explains: "I couldn't find any 'Grab' transactions. Did you perhaps use a different name like 'GrabFood'?"

---

## UC-INT-003: Multi-Source Investment Synthesis (FPT Research)

### 1. Summary

User asks for a "Bullish vs. Bearish" take on FPT stock to decide on a buy.

### 2. Actors

- **Primary Actor**: AI-First Investor (User)
- **Supporting Systems**: Technical Analysis API, News Analyze API.

### 3. Basic Flow

1.  User types: "Tell me about FPT's current sentiment."
2.  System triggers TWO tools:
    - `getTickerTechnicals(ticker="FPT")` -> Returns Bullish (MA Cross), RSI 65.
    - `getNewsSentiment(ticker="FPT")` -> Returns Neutral-Bullish (Positive Earnings, Slow Cloud sector).
3.  AI synthesizes the data points into a single "Wealth Report":
    - "Technically: FPT is in a strong uptrend. Price > MA50."
    - "Fundamentally: News sentiment is +0.4 on solid quarterly growth."
    - "Summary: Bullish. A good entry point if 1D RSI pulls back to 60."
4.  AI renders a "Live Price Chip" next to the text.

### 4. Postconditions

- User has a high-fidelity, multi-source research summary in seconds.
