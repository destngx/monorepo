import type { FinancialHealth, HealthComponent, NetWorthProjection } from './api/types';
import { formatPercent, formatSignedPercent } from './format';

export const DEBT_TO_INCOME_EXPLANATION =
  'Monthly debt payments / monthly income: a 5% minimum on each credit card balance, over the average monthly income. Money lent is an asset, not a debt payment.';

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

type Pillar = Pick<HealthComponent, 'key' | 'value' | 'target' | 'unit'>;

/** A pillar's measured value in words: "3.2 months", "+10.0%", "4 of 6 months". */
export function pillarValue(c: Pillar): string {
  if (c.key === 'spendingTrend') return formatSignedPercent(c.value);
  if (c.unit === '%') return formatPercent(c.value);
  if (c.unit === 'months') return `${c.value.toFixed(1)} months`;
  return `${Number(c.value.toFixed(2))} ${c.unit}`;
}

/** What a pillar needs for full marks. */
export function pillarTarget(c: Pillar): string {
  if (c.key === 'spendingTrend') return 'flat or falling';
  if (c.unit === '%') return formatPercent(c.target, 0);
  if (c.unit === 'months') return `${c.target} months`;
  return `${Number(c.target.toFixed(2))} ${c.unit}`;
}

// Short names for the radar's axes, so eight labels fit around it.
const AXIS_NAMES: Record<string, string> = {
  savings: 'Savings',
  consistency: 'Surplus',
  emergency: 'Cash',
  needs: 'Needs',
  debt: 'Debt',
  spendingTrend: 'Trend',
  lending: 'Lent',
  diversification: 'Spread',
};

/** The radar axis label of a pillar; an unknown key keeps its own label. */
export function axisName(c: Pick<HealthComponent, 'key' | 'label'>): string {
  return AXIS_NAMES[c.key] ?? c.label;
}
