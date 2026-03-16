import { LOC_PHAT_TAI_IDENTITY } from './prompts';
import { getAccounts } from '../sheets/accounts';
import { getBudget } from '../sheets/budget';
import { getExchangeRate } from '../services/exchange-rate-service';
import { formatVND, convertUSDTtoVND } from '../utils/currency';

export { LOC_PHAT_TAI_IDENTITY };

export async function buildSystemPrompt(taskInstruction?: string): Promise<string> {
  const [accounts, budget, rateResult] = await Promise.all([
    getAccounts().catch(() => []),
    getBudget().catch(() => []),
    getExchangeRate().catch(() => 25000)
  ]);

  const rate = typeof rateResult === 'number' ? rateResult : 25000;
  const currentDate = new Date().toLocaleDateString('vi-VN');

  const totalNetWorthVND = accounts.reduce((acc, a) => {
    return acc + (a.currency === 'USDT' || a.currency === 'USD' ? convertUSDTtoVND(a.balance, rate) : (a.balance || 0));
  }, 0);

  const accountSummary = accounts.map(a => `${a.name}: ${a.currency === 'VND' ? formatVND(a.balance || 0) : (a.balance || 0) + ' ' + a.currency}`).join(', ') || 'No accounts synced.';
  const budgetSummary = budget.filter(b => b.monthlyLimit > 0).map(b => `${b.category}: ${formatVND(b.monthlySpent || 0)} / ${formatVND(b.monthlyLimit)}`).join(', ') || 'No budget info set.';

  let prompt = `${LOC_PHAT_TAI_IDENTITY}

---

## LIVE DATA SNAPSHOT (Auto-injected)
- **Today**: ${currentDate}
- **Total Net Worth**: ${formatVND(totalNetWorthVND)}
- **Account Balances**: ${accountSummary}. Note: Binance balance may be in VND total, it means we already multiple by (banking) exchange rate.
- **Monthly Budget**: ${budgetSummary}

---

## OPERATIONAL DIRECTIVE
- ALWAYS use tools to respond to queries about the user's specific data, spending, history, or categorization
- Use **webSearch** for any question requiring real-time external data (prices, news, rates, events)
- If tools fail or return empty, clearly state the limitation and use your domain expertise as fallback
- Maintain your 7 operational identities in every interaction — synthesize across domains when answering`;

  if (taskInstruction) {
    prompt += `\n\n---\n\n## TASK-SPECIFIC CONTEXT\n${taskInstruction}`;
  }

  return prompt;
}
