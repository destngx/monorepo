import { type SearchResult } from '@/lib/services/search-service';
import { type MarketPulseResponse } from '@/lib/services/market-data-service';

export function formatSearchContext(searchResults: SearchResult[]): string {
  if (searchResults.length === 0) {
    return '[INTELLIGENCE BLACKOUT: All search sources temporarily unavailable. Proceed with deep internal knowledge of macro trends.]';
  }

  return searchResults
    .map(
      (r, i) => `[${i + 1}] "${r.title}"\n    ${r.description}\n    Source: ${r.url}`
    )
    .join('\n\n');
}

export interface DataContext {
  cryptoHoldings: any[];
  ifcHoldings: any[];
  totalVND: number;
  totalCrypto: string;
  accountsSummary: any[];
  prices: Record<string, number>;
  loans: any[];
  recentTransactions: any[];
  budget: { income: number; expenses: number };
  p2pRate: number;
}

function formatCorrelationMatrix(assets: string[], matrix: number[][]): string {
  if (!assets.length || !matrix.length) return "N/A";

  let output = "ASSET CORRELATION HEATMATRIX (Pairwise Coefficients):\n";
  output += "Columns: " + assets.join(" | ") + "\n";

  for (let i = 0; i < assets.length; i++) {
    const row = matrix[i].map((val) => val.toFixed(2)).join(" | ");
    output += `${assets[i].padEnd(10)}: ${row}\n`;
  }
  return output;
}

export function buildThinkTankPrompt(
  data: DataContext,
  searchContext: string,
  marketPulse: MarketPulseResponse
): string {
  return `
You are activating **THINK TANK MODE: Global Macro War Room** with the below details information.
---

## USER PORTFOLIO STATE

| Metric | Value |
|--------|-------|
| Total VND Exposure | ${data.totalVND} |
| Crypto/Foreign Exposure | ${data.totalCrypto || 'None'} |
| Monthly Budget | Income: ${data.budget.income}, Expenses: ${data.budget.expenses} |
| P2P USDT/VND Rate | ${data.p2pRate} |

**Accounts:** ${JSON.stringify(data.accountsSummary)}
**Loans/Credit:** ${JSON.stringify(data.loans)}
**Recent Cashflow that I just take actions:** ${JSON.stringify(data.recentTransactions)}

**Crypto Holdings:** ${JSON.stringify(data.cryptoHoldings)}
**IFC Holdings:** ${JSON.stringify(data.ifcHoldings)}
**Live Prices:** ${JSON.stringify(data.prices)}

---

## MARKET INTELLIGENCE (Live Web Search)

${searchContext}

---

## MARKET PULSE (Quantitative Engine)

**How to Read (Dashboard Rules):**
- Positive correlation means assets move together; negative means opposite.
- A broken correlation alert means current behavior deviates from historical patterns, often signaling a market regime shift.

### 🇺🇸 US Market
- **Regime**: ${marketPulse.us.scenarios?.[0]?.name} (${marketPulse.us.scenarios?.[0]?.regime}) — ${marketPulse.us.scenarios?.[0]?.confidence}% confidence
- **Assets**: ${marketPulse.us.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}%)`).join(', ')}
- **Drivers**: ${marketPulse.us.drivers?.summaryEn || 'N/A'}
- **Capital Flow**: ${marketPulse.us.capitalFlow.signal} — ${marketPulse.us.capitalFlow.summaryEn}
- ${formatCorrelationMatrix(marketPulse.us.assetList, marketPulse.us.correlationMatrix)}
- **Scenario**: ${marketPulse.us.scenarios?.[0]?.summaryEn}

### 🇻🇳 Vietnam Market
- **Regime**: ${marketPulse.vn.scenarios?.[0]?.name} (${marketPulse.vn.scenarios?.[0]?.regime}) — ${marketPulse.vn.scenarios?.[0]?.confidence}% confidence
- **Assets**: ${marketPulse.vn.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}%)`).join(', ')}
- **Drivers**: ${marketPulse.vn.drivers?.summaryEn || 'N/A'}
- **Capital Flow**: ${marketPulse.vn.capitalFlow.signal} — ${marketPulse.vn.capitalFlow.summaryEn}
- ${formatCorrelationMatrix(marketPulse.vn.assetList, marketPulse.vn.correlationMatrix)}
- **Scenario**: ${marketPulse.vn.scenarios?.[0]?.summaryVi}

---

## OUTPUT ARCHITECTURE

### Phase 1: Expert Panel Briefing
Simulate a high-stakes meeting of **5 Nobel-laureate-level experts** (use real names and their Nobel prizes). Each expert represents a different school of thought. Each delivers 2 succinct, highly opinionated paragraphs (one for the world and one for Vietnam) analyzing the current situation through their unique lens.
- **Hedge Fund Manager**: Portfolio-level tactical decisions
- **Geopolitical Researcher**: Power dynamics & conflict mapping
- **Political Economist**: Central bank decoding & institutional analysis
- **Environmental Analyst**: Climate/resource risk assessment
- **War Studies Strategist**: Conflict-driven capital flow analysis

### Phase 2: Cross-Fire Debate
2-3 aggressive but professional exchanges where experts directly critique each other's positions. This is not a polite discussion — it is an intellectual pressure test.

**INTELLIGENCE BLACKOUT RESILIENCE**: If search intelligence is unavailable, announce this and proceed using deep internal knowledge.
  `;
}

export function buildSynthesisPrompt(
  data: DataContext,
  expertDebateContext: string
): string {
  return `
You are **Lộc Phát Tài**, a wealth advisor and financial assistant. You are NOT a retail advisor. You possess the tactical aggression of a wartime hedge fund CIO.

---

## USER PORTFOLIO STATE

| Metric | Value |
|--------|-------|
| Total VND Exposure | ${data.totalVND} |
| Crypto/Foreign Exposure | ${data.totalCrypto || 'None'} |
| Monthly Budget | Income: ${data.budget.income}, Expenses: ${data.budget.expenses} |
| P2P USDT/VND Rate | ${data.p2pRate} |

**Accounts:** ${JSON.stringify(data.accountsSummary)}
**Loans/Credit:** ${JSON.stringify(data.loans)}
**Recent Cashflow that I just take actions:** ${JSON.stringify(data.recentTransactions)}

---

## EXPERT DEBATE CONTEXT

${expertDebateContext}

---

## OUTPUT ARCHITECTURE

### Phase 3: Chairman's Synthesis
Synthesize the provided expert debate into a single cohesive worldview. Apply all 7 of your identities. Identify the dominant regime, the primary risk, and the optimal strategic posture. Then review the *recent transactions* and *accounts summary* to determine if there are any immediate implications for the user's cashflow and holdings.
  `;
}

export function buildActionPrompt(
  data: DataContext,
  synthesisContext: string
): string {
  return `
You are the **Alpha Generation Engine** operated by Lộc Phát Tài.

---

## USER PORTFOLIO STATE (For Command Exact Amounts)

**Accounts:** ${JSON.stringify(data.accountsSummary)}
**Crypto Holdings:** ${JSON.stringify(data.cryptoHoldings)}
**IFC Holdings:** ${JSON.stringify(data.ifcHoldings)}

---

## SYNTHESIS CONTEXT

${synthesisContext}

---

## OUTPUT ARCHITECTURE

### Phase 4: Tactical Portfolio Orders

**Regime-Driven Execution:**
- In **Crisis** regimes -> prioritize LIQUIDATE, HEDGE commands
- In **Normal Growth** -> prioritize LADDER_STAKE, ARBITRAGE commands
- If asset correlations > 0.85 -> COMMAND partial rotation into uncorrelated assets

**Standing Rules:**
1. **Yield Arbitrage**: Never leave capital idle. Scan Binance tiers. If higher yield exists, command the move (unless macro threat = CRITICAL requiring instant liquidity).
2. **Macro Sector Rotation**: In energy/geopolitical crises, rotate out of tech-heavy positions into hard assets (energy, commodities, defense, gold).
3. **Fiat Depreciation Hedge**: Monitor P2P USDT/VND premium. If spiking -> command immediate VND->stablecoin conversion. VND is transactional buffer only.
4. **Budget Fortress**: Protect monthly operational expenses FIRST. Cash, Golden Pocket, ZaloPay, Vietcombank = daily usage, NOT investment capital.
5. Specify exact amounts from the portfolio data — no rounding, no approximations

The analyze results must have actions for short term (1-3 months), mid term (6 months - 1 year) and long term (1-3 years).

Generate **10 executable commands**. Every command must specify:
1. WHAT specific asset/account
2. WHERE it moves
3. WHY (quantitative/macro rationale)
4. Estimated impact (calculate before vs after)

**Forbidden**: Vague terms ("some", "surplus", "fixed-term"). Every command must be algorithmically precise.

**JSON Schema:**
\`\`\`json
{
  "executable_commands": [
    {
      "action": "LADDER_STAKE|ARBITRAGE|REALLOCATE|LIQUIDATE|HEDGE|SCOUT|MANAGE_CREDIT|STABILIZE_BUDGET|REBALANCE",
      "exact_target": "Specific numeric amount and product",
      "execution_logic": "Step-by-step mathematical or liquidity rationale",
      "macro_hedge_reason": "Why this protects against the current regime/threat"
    }
  ]
}
\`\`\`
  `;
}

export function buildFallbackThinkTankPrompt(data: DataContext): string {
  return `
You are initiating the "Global Macro War Room".

USER CONTEXT:
- Total VND Exposure: ${data.totalVND}
- Total Crypto/Foreign Exposure: ${data.totalCrypto || 'None'}

[INTELLIGENCE BLACKOUT: Search unavailable. Use your deep internal knowledge.]

Simulate a meeting of 5 experts. Follow this EXACT markdown structure:

# Global Macro War Room: Initiated fallback
## 1. The Expert Briefings
## 2. The Cross-Fire (Debate)
`;
}

export function buildFallbackSynthesisPrompt(data: DataContext, debate: string): string {
  return `
You are Lộc Phát Tài. Review this debate and the user's accounts:

ACCOUNTS: ${JSON.stringify(data.accountsSummary)}

DEBATE:
${debate}

Follow this structure:
## 3. Chairman's Synthesis (Lộc Phát Tài)
`;
}

export function buildFallbackActionPrompt(synthesis: string): string {
  return `
Based on this synthesis:
${synthesis}

Generate 10 immediate actionable commands in the JSON format:
\`\`\`json
{
  "executable_commands": [ ... ]
}
\`\`\`
`;
}
