import type { IdleCash, IdleCashVerdict } from '$lib/api/types';
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

/** The lines that take money off spendable cash, in the order of the math. */
export function keepLiquidLines(idle: IdleCash): { label: string; amount: number }[] {
  const c = idle.components;
  const months = c.bufferMonths === 1 ? '1 month' : `${Number(c.bufferMonths.toFixed(1))} months`;
  return [
    { label: 'Bills due in the next 30 days', amount: c.billsDue },
    { label: 'Card payments due in the next 30 days', amount: c.cardPayments },
    { label: 'Goal set-aside this month', amount: c.goalSetAside },
    { label: `Spending buffer (${months} of average expense)`, amount: c.spendingBuffer },
    {
      label: c.emergencyFromGoal ? 'Emergency fund target' : 'Emergency fund (months of needs)',
      amount: c.emergencyFund,
    },
  ];
}

/** The accounts assumed to hold the idle cash (lowest rate first). */
export function idleAccounts(idle: IdleCash) {
  return idle.accounts.filter((account) => account.idle > 0);
}
