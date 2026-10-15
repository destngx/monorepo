import type {
  CardBankReport,
  CardPaymentQuery,
  CardPeriodHistory,
  CardReport,
  CashbackAmount,
  MccRange,
  MccStatus,
  PeriodStanding,
  ReconciliationStatus,
  UtilizationBand,
} from '$lib/api/types';

export function allCards(banks: CardBankReport[]): CardReport[] {
  return banks.flatMap((bank) => bank.cards);
}

/** Cards to list: closed cards only when asked for. */
export function visibleCards(cards: CardReport[], showClosed: boolean): CardReport[] {
  return showClosed ? cards : cards.filter((card) => !card.closedFrom);
}

export type CardTagTone = 'green' | 'blue' | 'violet' | 'amber' | 'orange' | 'neutral';

const CARD_TAG_TONES: CardTagTone[] = ['green', 'blue', 'violet', 'amber', 'orange'];

/**
 * Card tags get their card's colour (WM-TX-3). A tag belongs to the card whose
 * keyword it contains (the longest one wins, as "rose gold" over "gold"); tones
 * follow the sorted keywords, so every page that knows the same cards agrees.
 */
export function cardTagTone(tag: string, keywords: string[]): CardTagTone {
  const sorted = [...new Set(keywords.map((keyword) => keyword.trim().toLowerCase()).filter(Boolean))].sort();
  const lower = tag.toLowerCase();
  let match = -1;
  sorted.forEach((keyword, index) => {
    if (lower.includes(keyword) && (match < 0 || keyword.length > sorted[match].length)) match = index;
  });
  return match < 0 ? 'neutral' : CARD_TAG_TONES[match % CARD_TAG_TONES.length];
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

/** Totals across banks: limits and usage are per bank, as the bank's cards share them (WM-CARD-16). */
export function portfolioTotals(banks: CardBankReport[]) {
  const sum = (pick: (bank: CardBankReport) => number) => banks.reduce((total, bank) => total + pick(bank), 0);
  const confirmed = sum((bank) => bank.cashback.confirmed);
  const total = sum((bank) => bank.cashback.total);
  return {
    limit: sum((bank) => bank.limit),
    usage: sum((bank) => bank.usage),
    cashback: { confirmed, estimated: total - confirmed, total } satisfies CashbackAmount,
    refundYtd: sum((bank) => bank.refundYtd),
    feesPaidYtd: sum((bank) => bank.feesPaidYtd),
    netEarnYtd: sum((bank) => bank.netEarnYtd),
  };
}

/**
 * Cashback cap per counting period for a card: the programme's card-level cap
 * when set, else the sum of the rule caps when every earning rule is capped;
 * null when some rule is uncapped (no ceiling). Spend tiers of one rule never
 * pay together, so only the largest tier cap counts; a cap group's rules share
 * one cap, so it counts once; exclusions earn nothing.
 */
export function cardPeriodCap(card: Pick<CardReport, 'cap' | 'rules'>): number | null {
  if (card.cap > 0) return card.cap;
  const earning = card.rules.filter((rule) => !rule.exclude);
  if (!earning.length || earning.some((rule) => !(rule.cap > 0))) return null;
  const tierCaps = new Map<string, number>();
  for (const rule of earning) {
    const key = rule.capGroup
      ? JSON.stringify(['group', rule.capGroup])
      : JSON.stringify([formatMccs(rule.mccs), !!rule.fallback, rule.minTransaction]);
    tierCaps.set(key, Math.max(tierCaps.get(key) ?? 0, rule.cap));
  }
  return [...tierCaps.values()].reduce((sum, cap) => sum + cap, 0);
}

/** Room left under a cap this period. */
export interface CapLeft {
  name: string;
  left: number;
  cap: number;
}

/**
 * What still bounds this period's cashback: the card-level cap when the
 * programme has one, else each capped rule or cap group. Empty when nothing
 * is capped.
 */
export function capsLeft(standing: PeriodStanding): CapLeft[] {
  if (standing.cap > 0) return [{ name: 'Card', left: standing.capLeft, cap: standing.cap }];
  return standing.rules
    .filter((rule) => rule.cap > 0)
    .map((rule) => ({ name: rule.name, left: rule.capLeft, cap: rule.cap }));
}

type Tone = 'neutral' | 'green' | 'amber' | 'red' | 'blue';

/** How a period's refund compares with its cashback (WM-CARD-11). */
export const RECONCILIATION: Record<ReconciliationStatus, { label: string; tone: Tone; hint: string }> = {
  open: { label: 'Open', tone: 'neutral', hint: 'The period has not ended.' },
  awaiting: { label: 'Awaiting refund', tone: 'blue', hint: 'The period ended and no refund is recorded yet.' },
  match: { label: 'Matches', tone: 'green', hint: 'The refund lies between the confirmed and the estimated cashback.' },
  below: {
    label: 'Below',
    tone: 'red',
    hint: 'The refund is less than the confirmed cashback: check the statement with the bank.',
  },
  above: {
    label: 'Above',
    tone: 'amber',
    hint: 'The refund exceeds the estimate: an assumed or unknown MCC is likely wrong.',
  },
};

/** How sure the engine is of a payment's MCC (C3, C4). */
export const MCC_STATUS: Record<MccStatus, { label: string; tone: Tone }> = {
  confirmed: { label: 'confirmed', tone: 'green' },
  assumed: { label: 'assumed', tone: 'amber' },
  unknown: { label: 'unknown MCC', tone: 'neutral' },
};

/** MCC ranges in the sheet's notation, e.g. "0742, 5812-5814". */
export function formatMccs(ranges: MccRange[] | undefined): string {
  const code = (value: number) => String(value).padStart(4, '0');
  return (ranges ?? []).map((r) => (r.from === r.to ? code(r.from) : `${code(r.from)}-${code(r.to)}`)).join(', ');
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

export type CardMetric = 'refund' | 'cashback' | 'spent' | 'efficiency';

const METRIC_VALUE: Record<CardMetric, (period: CardPeriodHistory) => number> = {
  refund: (period) => period.refund,
  cashback: (period) => period.cashback.total,
  spent: (period) => period.spent,
  efficiency: (period) => period.efficiency,
};

/**
 * Cross-card comparison for one year (WM-CARD-2): the periods any selected
 * card has history in, by the month they start in, oldest first, and one series per card with null for
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
      const byMonth = new Map(card.history.map((m) => [m.month, METRIC_VALUE[metric](m)]));
      return { name: card.name, values: months.map((month) => byMonth.get(month) ?? null) };
    }),
  };
}

/** Reads the best-card form (WM-CARD-13): a positive amount and the payee, the MCC or both. */
export function parseCardPayment(form: {
  payee: string;
  amount: string;
  mcc: string;
}): { query: CardPaymentQuery } | { error: string } {
  const amount = Number(form.amount.replaceAll(',', '').trim());
  if (!form.amount.trim() || !Number.isFinite(amount) || amount <= 0) return { error: 'Enter a positive amount.' };
  const mccText = form.mcc.trim();
  if (mccText && !/^\d{4}$/.test(mccText)) return { error: 'An MCC is 4 digits, e.g. 5411.' };
  const payee = form.payee.trim();
  if (!payee && !mccText) return { error: 'Enter the payee or its MCC.' };
  return { query: { payee, amount, ...(mccText ? { mcc: Number(mccText) } : {}) } };
}
