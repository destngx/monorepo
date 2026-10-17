import { describe, expect, it } from 'vitest';
import {
  IDLE_CASH_VERDICT,
  depositKind,
  depositTerms,
  idleAccounts,
  keepLiquidLines,
  keepReady,
  parseFloatDraft,
  rateLabel,
} from './idleCash';
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

  it('Given emergency accounts and a default float When listing what stays Then the uncovered target and the float are named', () => {
    const lines = keepLiquidLines(idleCash());
    expect(lines.map((line) => line.amount)).toEqual([8_000_000, 2_000_000, 5_000_000, 10_000_000, 35_000_000]);
    expect(lines[3].label).toBe('Emergency fund target, the part not in emergency accounts');
    expect(lines[4].label).toBe('Day-to-day float (1.5 months of average expense)');
    const other = idleCash();
    other.components = {
      ...other.components,
      emergencyFromGoal: false,
      emergencyCovered: 0,
      floatFromPreference: true,
    };
    const lines2 = keepLiquidLines(other);
    expect(lines2[3].label).toBe('Emergency fund (months of needs)');
    expect(lines2[4].label).toBe('Day-to-day float (your setting)');
  });

  it('keeps ready what is not idle', () => {
    expect(keepReady(idleCash())).toBe(60_000_000);
    expect(keepReady(idleCash({ idle: 200_000_000 }))).toBe(0);
  });

  it('Given a 1-month or no-term best When wording it Then the kind and terms never leave empty parentheses', () => {
    const best = idleCash().best!;
    expect(depositKind(best)).toBe('a 1-month deposit');
    expect(depositTerms(best)).toBe('rolls monthly; withdraw early at the no-term rate');
    const noTerm = { ...best, access: 'no_term' as const, tenor: 'no-term', tenorMonths: 0 };
    expect(depositKind(noTerm)).toBe('a no-term deposit');
    expect(depositTerms(noTerm)).toBe('withdraw any time');
  });

  it('keeps only the accounts holding idle cash', () => {
    expect(idleAccounts(idleCash()).map((account) => account.name)).toEqual(['Account A', 'Account B']);
  });

  it('reads a typed float with Vietnamese grouping and rejects anything else', () => {
    expect(parseFloatDraft('15.000.000')).toEqual({ value: 15_000_000, error: null });
    expect(parseFloatDraft(' 15 000 000 ')).toEqual({ value: 15_000_000, error: null });
    expect(parseFloatDraft('0')).toEqual({ value: 0, error: null });
    expect(parseFloatDraft('').error).toBeTruthy();
    expect(parseFloatDraft('-5').error).toBeTruthy();
    expect(parseFloatDraft('1e9').error).toBeTruthy();
    expect(parseFloatDraft('999999999999').error).toBeTruthy();
  });
});
