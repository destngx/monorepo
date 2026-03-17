import { readSheet } from '../sheets/client';
import { getLoans } from '../sheets/loans';
import { getTransactions } from '../sheets/transactions';
import { getBudget } from '../sheets/budget';
import { getExchangeRate } from '../exchange-rate-service';
import { type DataContext } from '../../ai/prompts/market/investment';

/**
 * Investment Data Service: Extracts and prepares structured investment context.
 */
export async function extractInvestmentData(accounts: any[]): Promise<DataContext> {
  const totalVND = accounts
    .filter((a: any) => a.currency === 'VND')
    .reduce((s: number, a: any) => s + (a.balance || 0), 0);

  const totalCrypto = accounts
    .filter((a: any) => ['BTC', 'ETH', 'SOL', 'USDT', 'USDC'].includes(a.currency))
    .map((a: any) => `${a.balance} ${a.currency}`)
    .join(', ');

  const [cryptoSheetData, ifcSheetData, loans, transactions, budgetItems, p2pRate] = await Promise.all([
    readSheet('Crypto!A1:Z100').catch(() => []),
    readSheet('InvestmentFundCertificate!A1:Z100').catch(() => []),
    getLoans().catch(() => []),
    getTransactions().catch(() => []),
    getBudget().catch(() => []),
    getExchangeRate().catch(() => 25400),
  ]);

  const parseForAI = (data: any[], keywords: string[]) => {
    const rows = data as string[][];
    const headerIdx = rows.findIndex((r) => r.some((c) => keywords.includes(c?.trim())));
    if (headerIdx === -1) return [];

    const headersRaw = rows[headerIdx];
    const headers = headersRaw.map((h: any, i: any) => (i === 0 && !h ? 'Platform' : h || `Col${i}`));
    let lastProv = '';

    return rows
      .slice(headerIdx + 1)
      .map((row) => {
        const obj: any = {};
        if (row[0]) lastProv = row[0];
        headers.forEach((h, i) => {
          if (h) obj[h] = i === 0 ? lastProv : row[i] || '';
        });
        return obj;
      })
      .filter((o) => {
        return Object.entries(o).some(([k, v]) => k !== 'Platform' && v !== '');
      });
  };

  const cryptoHoldings = parseForAI(cryptoSheetData, ['Currency', 'Spot/Fund', 'Token']);
  const ifcHoldings = parseForAI(ifcSheetData, ['Index', 'Total Unit', 'Certificate']);

  const accountsSummary = accounts.map((a: any) => ({
    name: a.name,
    balance: a.balance,
    currency: a.currency,
    type: a.type,
  }));

  const recentTransactions = transactions.slice(-15).map((t) => ({
    date: t.date,
    payee: t.payee,
    amount: t.payment ? `-${t.payment}` : `+${t.deposit}`,
    category: t.category,
    account: t.accountName,
  }));

  const now = new Date();
  const currentMonthKey = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;

  const budget = { income: 0, expenses: 0 };
  for (const item of budgetItems) {
    const limit = item.monthlyLimits?.[currentMonthKey] || item.monthlyLimit || 0;
    if (item.categoryType === 'income') budget.income += limit;
    else if (item.categoryType === 'expense') budget.expenses += limit;
  }

  return {
    cryptoHoldings,
    ifcHoldings,
    totalVND,
    totalCrypto,
    accountsSummary,
    prices: {},
    loans,
    recentTransactions,
    budget,
    p2pRate,
  };
}

export function buildSearchQueries(data: DataContext): string[] {
  const tokens: string[] = [];

  for (const h of data.cryptoHoldings) {
    if (h.Currency || h.Token) tokens.push(h.Currency || h.Token);
  }
  for (const h of data.ifcHoldings) {
    if (h.Certificate || h.Index) tokens.push(h.Certificate || h.Index);
  }
  for (const a of data.accountsSummary) {
    if (['BTC', 'ETH', 'SOL'].includes(a.currency) && !tokens.includes(a.currency)) {
      tokens.push(a.currency);
    }
  }

  const now = new Date();
  const monthYear = now.toLocaleString('en-US', { month: 'long', year: 'numeric' });
  const uniqueTokens = [...new Set(tokens)].slice(0, 5).join(' ');

  const queries = [
    `Binance Flexible Earn APY interest rates ${monthYear} USDT USDC ETH SOL`,
    `World Macro Breaking News Geopolitics FED interest rate ${monthYear}`,
  ];

  if (tokens.length === 0) {
    queries.push(`Vietnam SBV state bank policy status ${monthYear}`);
  } else {
    queries.push(`crypto forecast ${uniqueTokens} ${monthYear}`);
  }

  return queries;
}
