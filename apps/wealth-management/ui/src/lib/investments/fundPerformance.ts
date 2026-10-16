import type { FundCostStatus, FundPerformance, FundReturn, VsDeposit } from '$lib/api/types';
import type { StatusTone } from '$lib/status';

// The engine computes fund performance and the deposit comparison (ROADMAP M3,
// WM-INV-21/22); this module only words it.

/** Ahead or behind a deposit, as a status with its own label (never colour alone). */
export const VS_DEPOSIT: Record<VsDeposit, { tone: StatusTone; label: string }> = {
  ahead: { tone: 'good', label: 'Ahead of deposit' },
  behind: { tone: 'warning', label: 'Behind deposit' },
  even: { tone: 'neutral', label: 'About even' },
};

/** Why a holding shows its value only. */
export const COST_STATUS_LABEL: Record<FundCostStatus, string> = {
  known: 'Cost known',
  no_history: 'Cost unknown: no History',
  history_invalid: 'Cost unknown: History unreadable',
  units_mismatch: 'Cost unknown: History ≠ Total Unit',
  nav_unavailable: 'Cost unknown: no NAV history',
};

/** The headline case: what the total card can say first. */
export type FundHeadline =
  | { kind: 'empty' }
  | { kind: 'no_cost' }
  | { kind: 'no_deposit'; total: FundReturn; issue: string }
  | { kind: 'compare'; total: FundReturn; rate: number; vs: VsDeposit };

export function fundHeadline(performance: FundPerformance): FundHeadline {
  const { totals, deposit } = performance;
  if (!totals.holdings) return { kind: 'empty' };
  const total = totals.return;
  if (!total) return { kind: 'no_cost' };
  if (!deposit || total.depositValue === null || !total.vsDeposit) {
    return { kind: 'no_deposit', total, issue: performance.depositIssue ?? 'no deposit rate' };
  }
  return { kind: 'compare', total, rate: deposit.rate, vs: total.vsDeposit };
}

/** "Bank A · 12M · 5.10% a year", the rate the comparison used. */
export function depositRateLabel(performance: FundPerformance): string | null {
  const deposit = performance.deposit;
  if (!deposit) return null;
  return [deposit.bank, deposit.tenor, `${deposit.rate.toFixed(2)}% a year`].filter(Boolean).join(' · ');
}

/** "+1.25 points a year" for an annualised difference. */
export function formatPoints(points: number | null): string {
  if (points === null || !Number.isFinite(points)) return '-';
  return `${points > 0 ? '+' : ''}${points.toFixed(2)} points a year`;
}
