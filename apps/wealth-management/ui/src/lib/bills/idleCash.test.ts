import { describe, expect, it } from 'vitest';
import { IDLE_CASH_VERDICT, idleAccounts, keepLiquidLines, rateLabel } from './idleCash';
import { idleCash } from './idleCash.testdata';

describe('idle cash helpers', () => {
  it('gives every verdict a status and a label', () => {
    expect(IDLE_CASH_VERDICT.worth_moving).toEqual({ tone: 'warning', label: 'Worth moving' });
    expect(IDLE_CASH_VERDICT.nothing_idle.tone).toBe('good');
    expect(IDLE_CASH_VERDICT.no_rate.tone).toBe('neutral');
    expect(IDLE_CASH_VERDICT.not_worth_it.label).toBe('Not worth moving');
  });

  it('formats rates without trailing zeros', () => {
    expect(rateLabel(4.5)).toBe('4.5%');
    expect(rateLabel(0.225)).toBe('0.23%');
    expect(rateLabel(0)).toBe('0%');
    expect(rateLabel(Number.NaN)).toBe('-');
  });

  it('lists what stays liquid, naming where the emergency fund comes from', () => {
    const lines = keepLiquidLines(idleCash());
    expect(lines.map((line) => line.amount)).toEqual([8_000_000, 2_000_000, 5_000_000, 15_000_000, 30_000_000]);
    expect(lines[3].label).toBe('Spending buffer (1.5 months of average expense)');
    expect(lines[4].label).toBe('Emergency fund target');
    const fromSettings = idleCash();
    fromSettings.components = { ...fromSettings.components, emergencyFromGoal: false, bufferMonths: 1 };
    const other = keepLiquidLines(fromSettings);
    expect(other[3].label).toBe('Spending buffer (1 month of average expense)');
    expect(other[4].label).toBe('Emergency fund (months of needs)');
  });

  it('keeps only the accounts holding idle cash', () => {
    expect(idleAccounts(idleCash()).map((account) => account.name)).toEqual(['Account A', 'Account B']);
  });
});
