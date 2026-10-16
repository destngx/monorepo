import type { IdleCash } from '$lib/api/types';

/** An invented idle cash check for tests. */
export const idleCash = (overrides: Partial<IdleCash> = {}): IdleCash => ({
  asOf: '2026-10-17',
  idle: 90_000_000,
  components: {
    spendableCash: 150_000_000,
    billsDue: 8_000_000,
    cardPayments: 2_000_000,
    goalSetAside: 5_000_000,
    spendingBuffer: 15_000_000,
    bufferMonths: 1.5,
    averageMonthlyExpense: 10_000_000,
    emergencyFund: 30_000_000,
    emergencyFromGoal: true,
    keepLiquid: 60_000_000,
  },
  accounts: [
    { name: 'Account A', balance: 50_000_000, rate: 0, rateFromNote: false, idle: 50_000_000 },
    { name: 'Account B', balance: 100_000_000, rate: 0.5, rateFromNote: true, idle: 40_000_000 },
    { name: 'Account C', balance: 0, rate: 1, rateFromNote: true, idle: 0 },
  ],
  currentRate: 0.22,
  best: { bank: 'Bank X', tenor: '', rate: 5.5, basis: 'A proxy: the best savings rate published today.' },
  rateGapPoints: 5.28,
  extraPerYear: 4_752_000,
  verdict: 'worth_moving',
  reasons: ['About 90.000.000 VND is idle.'],
  minRateGapPoints: 0.5,
  minExtraPerYear: 200_000,
  keepLiquidRule: 'Kept liquid: bills, card payments, goal set-aside, buffer and emergency fund.',
  rateRule: 'Rates come from the account note.',
  thresholdRule: 'Worth moving from 0.5 points and 200.000 VND a year.',
  ...overrides,
});
