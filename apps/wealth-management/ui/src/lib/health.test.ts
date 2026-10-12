import { describe, expect, it } from 'vitest';
import { DEBT_TO_INCOME_EXPLANATION, pillarText, projectionRows, usesMonthlyDebtPayments } from './health';

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
