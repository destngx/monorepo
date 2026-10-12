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
