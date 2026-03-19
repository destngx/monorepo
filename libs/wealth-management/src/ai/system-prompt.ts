import { getIdentityPrompt } from './prompts';
import { getAccounts, getBudget, getExchangeRate } from '@wealth-management/services/server';
import { formatVND, convertUSDTtoVND } from '@wealth-management/utils';
import type { Account } from '@wealth-management/types/accounts';
import type { BudgetItem } from '@wealth-management/types/budget';

export async function buildSystemPrompt(taskInstruction?: string): Promise<string> {
  const results = await Promise.all([
    getIdentityPrompt(),
    getAccounts().catch((): Account[] => []),
    getBudget().catch((): BudgetItem[] => []),
    getExchangeRate().catch(() => 25000),
  ]);

  const identity = results[0];
  const accounts = results[1];
  const budget = results[2];
  const rateResult = results[3];

  const rate = typeof rateResult === 'number' ? rateResult : 25000;
  const currentDate = new Date().toLocaleDateString('vi-VN');

  const totalNetWorthVND = accounts.reduce((acc, a) => {
    return acc + (a.currency === 'USDT' || a.currency === 'USD' ? convertUSDTtoVND(a.balance, rate) : a.balance || 0);
  }, 0);

  const accountSummary =
    accounts
      .map(
        (a) => `${a.name}: ${a.currency === 'VND' ? formatVND(a.balance || 0) : (a.balance || 0) + ' ' + a.currency}`,
      )
      .join(', ') || 'No accounts synced.';
  const budgetSummary =
    budget
      .filter((b) => b.monthlyLimit > 0)
      .map((b) => `${b.category}: ${formatVND(b.monthlySpent || 0)} / ${formatVND(b.monthlyLimit)}`)
      .join(', ') || 'No budget info set.';

  let prompt = `${identity}

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
