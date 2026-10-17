import type { DepositRate, ExclusionReason, IdleCash, IdleCashVerdict } from '$lib/api/types';
import type { StatusTone } from '$lib/status';

/** Verdict badge: always an icon and a label, never colour alone. */
export const IDLE_CASH_VERDICT: Record<IdleCashVerdict, { tone: StatusTone; label: string }> = {
  worth_moving: { tone: 'warning', label: 'Worth moving' },
  not_worth_it: { tone: 'good', label: 'Not worth moving' },
  nothing_idle: { tone: 'good', label: 'Nothing idle' },
  no_rate: { tone: 'neutral', label: 'No rate to compare' },
};

/** A rate in percent a year, without trailing zeros: 4.5%, 0.25%, 0%. */
export function rateLabel(rate: number): string {
  if (!Number.isFinite(rate)) return '-';
  return `${Number(rate.toFixed(2))}%`;
}

function monthsLabel(months: number): string {
  return months === 1 ? '1 month' : `${Number(months.toFixed(1))} months`;
}

/** The lines taken off ready money, in the order of the math (D34). */
export function keepLiquidLines(idle: IdleCash): { label: string; amount: number }[] {
  const c = idle.components;
  const emergency = c.emergencyFromGoal ? 'Emergency fund target' : 'Emergency fund (months of needs)';
  return [
    { label: 'Bills due in the next 30 days', amount: c.billsDue },
    { label: 'Card payments due in the next 30 days', amount: c.cardPayments },
    { label: 'Goal set-aside this month', amount: c.goalSetAside },
    {
      label: c.emergencyCovered > 0 ? `${emergency}, the part not in emergency accounts` : emergency,
      amount: c.emergencyFromReady,
    },
    { label: floatLabel(idle), amount: c.dayToDayFloat },
  ];
}

/** Where the day-to-day float comes from. */
export function floatLabel(idle: IdleCash): string {
  const c = idle.components;
  return c.floatFromPreference
    ? 'Day-to-day float (your setting)'
    : `Day-to-day float (${monthsLabel(c.bufferMonths)} of average expense)`;
}

/** What stays in the everyday accounts: ready money minus the idle part. */
export function keepReady(idle: IdleCash): number {
  return Math.max(0, idle.components.readyMoney - idle.idle);
}

/** "a 1-month deposit" or "a no-term deposit". */
export function depositKind(best: DepositRate): string {
  return best.access === 'one_month' ? 'a 1-month deposit' : 'a no-term deposit';
}

/** Capitalises the first letter, to start a sentence. */
export function sentence(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** How the instant-access option behaves, for the parentheses after its rate. */
export function depositTerms(best: DepositRate): string {
  return best.access === 'one_month' ? 'rolls monthly; withdraw early at the no-term rate' : 'withdraw any time';
}

/** The accounts assumed to hold the idle cash (lowest rate first). */
export function idleAccounts(idle: IdleCash) {
  return idle.accounts.filter((account) => account.idle > 0);
}

/** Why an account is never idle. */
export const IDLE_EXCLUSION_LABELS: Record<ExclusionReason, string> = {
  crypto: 'crypto, kept for holding and trading',
  not_ready: 'marked not ready to use',
};

/** The largest day-to-day float the engine accepts (VND). */
export const MAX_DAY_TO_DAY_FLOAT = 100_000_000_000;

/**
 * Reads a float typed by the user: digits with optional `.`, `,` or space
 * grouping ("15.000.000", "15 000 000"); blank is not a value.
 */
export function parseFloatDraft(text: string): { value: number | null; error: string | null } {
  const trimmed = text.trim();
  if (!trimmed) return { value: null, error: 'Enter an amount in VND.' };
  const digits = trimmed.replace(/[.,\s]/g, '');
  if (!/^\d+$/.test(digits)) return { value: null, error: 'Use whole VND, for example 15.000.000.' };
  const value = Number(digits);
  if (value > MAX_DAY_TO_DAY_FLOAT) return { value: null, error: 'That is more than the engine accepts.' };
  return { value, error: null };
}
