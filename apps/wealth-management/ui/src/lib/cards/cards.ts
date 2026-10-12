import type { CardBankReport, CardReport, UtilizationBand } from '$lib/api/types';

export function allCards(banks: CardBankReport[]): CardReport[] {
  return banks.flatMap((bank) => bank.accounts.flatMap((account) => account.cards));
}

export function findCard(banks: CardBankReport[], name: string): CardReport | undefined {
  return allCards(banks).find((card) => card.name === name);
}

// WM-CARD-4: <30% green, <70% amber, else red.
export const UTILIZATION_COLOR: Record<UtilizationBand, string> = {
  good: '#10b981',
  fair: '#f59e0b',
  high: '#f43f5e',
};

export const UTILIZATION_BAR: Record<UtilizationBand, string> = {
  good: 'bg-emerald-500',
  fair: 'bg-amber-400',
  high: 'bg-rose-500',
};

/** Per-card totals across a bank report. */
export function portfolioTotals(banks: CardBankReport[]) {
  const accounts = banks.flatMap((bank) => bank.accounts);
  const sum = (pick: (a: (typeof accounts)[number]) => number) => accounts.reduce((total, a) => total + pick(a), 0);
  return {
    limit: sum((a) => a.limit),
    usage: sum((a) => a.usage),
    estimatedCashback: sum((a) => a.estimatedCashback),
    refundYtd: sum((a) => a.refundYtd),
    annualFees: sum((a) => a.annualFees),
    netEarnYtd: sum((a) => a.netEarnYtd),
  };
}

/**
 * Monthly cashback cap for a card (WM-CARD-2): the engine's card-level cap when
 * it sends one, else the sum of the rule caps when every rule is capped; null
 * when some rule is uncapped (no ceiling).
 */
export function cardMonthlyCap(card: Pick<CardReport, 'rules'> & { monthlyCap?: number | null }): number | null {
  if (card.monthlyCap && card.monthlyCap > 0) return card.monthlyCap;
  if (!card.rules.length || card.rules.some((rule) => !(rule.capMonthly > 0))) return null;
  return card.rules.reduce((sum, rule) => sum + rule.capMonthly, 0);
}

/** Running total of a series, e.g. YTD refunds month by month. */
export function cumulative(values: number[]): number[] {
  let total = 0;
  return values.map((value) => (total += value));
}

/** Years with history across the given cards, newest first. */
export function historyYears(cards: Pick<CardReport, 'history'>[]): string[] {
  return [...new Set(cards.flatMap((card) => card.history.map((m) => m.month.slice(0, 4))))].sort().reverse();
}

export type CardMetric = 'actualRefund' | 'potentialCashback' | 'spent' | 'efficiency';

/**
 * Cross-card comparison for one year (WM-CARD-2): the months any selected
 * card has history in, oldest first, and one series per card with null for
 * months that card has no row.
 */
export function compareCards(
  cards: Pick<CardReport, 'name' | 'history'>[],
  year: string,
  metric: CardMetric,
): { months: string[]; series: { name: string; values: (number | null)[] }[] } {
  const months = [
    ...new Set(cards.flatMap((card) => card.history.map((m) => m.month).filter((m) => m.startsWith(year)))),
  ].sort();
  return {
    months,
    series: cards.map((card) => {
      const byMonth = new Map(card.history.map((m) => [m.month, m[metric]]));
      return { name: card.name, values: months.map((month) => byMonth.get(month) ?? null) };
    }),
  };
}
