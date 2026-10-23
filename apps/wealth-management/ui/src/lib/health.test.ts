import { describe, expect, it } from 'vitest';
import {
  axisName,
  DEBT_TO_INCOME_EXPLANATION,
  pillarTarget,
  pillarText,
  pillarValue,
  projectionRows,
  usesMonthlyDebtPayments,
} from './health';

describe('debt-to-income wording (WM-HEA-2)', () => {
  const debt = { key: 'debt', label: 'Debt to income' };

  it('explains monthly debt payments when the engine sends a projection', () => {
    expect(usesMonthlyDebtPayments({ projection: [] })).toBe(true);
    expect(pillarText(debt, { projection: [] })).toEqual({
      label: 'Debt to income (monthly)',
      explanation: DEBT_TO_INCOME_EXPLANATION,
    });
  });

  it('keeps the engine label for older engines and other pillars', () => {
    expect(pillarText(debt, {})).toEqual({ label: 'Debt to income', explanation: null });
    expect(pillarText({ key: 'savings', label: 'Savings rate' }, { projection: [] })).toEqual({
      label: 'Savings rate',
      explanation: null,
    });
  });
});

describe('projectionRows (WM-RULE-12)', () => {
  it('orders horizons and tolerates a missing projection', () => {
    const rows = projectionRows([
      { years: 20, nominal: 3, real: 2 },
      { years: 5, nominal: 1, real: 1 },
    ]);
    expect(rows.map((r) => r.years)).toEqual([5, 20]);
    expect(projectionRows(undefined)).toEqual([]);
  });
});

describe('pillar wording (D44)', () => {
  const pillar = (key: string, value: number, target: number, unit: string) => ({ key, value, target, unit });

  it('says each value and what earns full marks in its own unit', () => {
    expect(pillarValue(pillar('emergency', 3.24, 6, 'months'))).toBe('3.2 months');
    expect(pillarTarget(pillar('emergency', 3.24, 6, 'months'))).toBe('6 months');
    expect(pillarValue(pillar('consistency', 4, 6, 'of 6 months'))).toBe('4 of 6 months');
    expect(pillarTarget(pillar('consistency', 4, 6, 'of 6 months'))).toBe('6 of 6 months');
    expect(pillarValue(pillar('needs', 65, 50, '%'))).toBe('65.0%');
    expect(pillarTarget(pillar('needs', 65, 50, '%'))).toBe('50%');
    expect(pillarValue(pillar('diversification', 0.42, 1 / 3, 'HHI'))).toBe('0.42 HHI');
    expect(pillarTarget(pillar('diversification', 0.42, 1 / 3, 'HHI'))).toBe('0.33 HHI');
  });

  it('signs the spending trend and asks for flat or falling spend', () => {
    expect(pillarValue(pillar('spendingTrend', 10, 0, '%'))).toBe('+10.0%');
    expect(pillarTarget(pillar('spendingTrend', 10, 0, '%'))).toBe('flat or falling');
  });
});

describe('axisName (D44)', () => {
  it('shortens each pillar for the radar and keeps an unknown label', () => {
    expect(axisName({ key: 'diversification', label: 'Diversification' })).toBe('Spread');
    expect(axisName({ key: 'consistency', label: 'Months in surplus' })).toBe('Surplus');
    expect(axisName({ key: 'other', label: 'Other' })).toBe('Other');
  });
});
