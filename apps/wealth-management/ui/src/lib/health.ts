import type { FinancialHealth, HealthComponent, NetWorthProjection } from './api/types';

export const DEBT_TO_INCOME_EXPLANATION =
  'Monthly debt payments / monthly income: loan monthly payments plus a 5% minimum on each credit card balance, over the average monthly income.';

/**
 * Engines that send `projection` compute debt-to-income as monthly debt
 * payments / monthly income (WM-HEA-2); older engines use liabilities over
 * annual income, so their own label and formula stand.
 */
export function usesMonthlyDebtPayments(health: Pick<FinancialHealth, 'projection'>): boolean {
  return Array.isArray(health.projection);
}

/** Label and plain-language explanation for a pillar. */
export function pillarText(
  component: Pick<HealthComponent, 'key' | 'label'>,
  health: Pick<FinancialHealth, 'projection'>,
): { label: string; explanation: string | null } {
  if (component.key === 'debt' && usesMonthlyDebtPayments(health)) {
    return { label: 'Debt to income (monthly)', explanation: DEBT_TO_INCOME_EXPLANATION };
  }
  return { label: component.label, explanation: null };
}

/** Projection horizons, shortest first. */
export function projectionRows(projection: NetWorthProjection[] | undefined): NetWorthProjection[] {
  return [...(projection ?? [])].sort((a, b) => a.years - b.years);
}
