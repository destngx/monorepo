import type { CardBankReport, CardReport } from '$lib/api/types';

// Invented card figures for page tests.
const amount = (total: number) => ({ confirmed: total, estimated: 0, total });

export const testCard = (overrides: Partial<CardReport> = {}): CardReport => ({
  name: 'Blue Rewards',
  bank: 'Harbor Bank',
  program: 'Everyday 5%',
  keyword: 'blue',
  accountName: 'Harbor Card',
  expiry: '12/29',
  annualFee: 499_000,
  cap: 300_000,
  rules: [],
  cycle: {
    statementDay: 20,
    dueDay: 5,
    cycleStart: '2026-09-21',
    cycleEnd: '2026-10-20',
    cycleDays: 30,
    cycleElapsedDays: 27,
    cycleProgress: 90,
    payBy: '2026-10-29',
    daysUntilPayBy: 12,
    payByUrgent: false,
  },
  standing: {
    start: '2026-09-21',
    end: '2026-10-20',
    spent: 6_420_000,
    qualifyingSpend: 6_420_000,
    cashback: amount(187_000),
    cap: 300_000,
    capLeft: 113_000,
    rules: [],
    paymentCount: 9,
    unconfirmedPayments: 0,
  },
  refundYtd: 1_310_000,
  feesPaidYtd: 0,
  netEarnYtd: 1_310_000,
  lifetimeFees: 499_000,
  history: [],
  transactions: [],
  ...overrides,
});

export const testBank = (cards: CardReport[] = [testCard()]): CardBankReport => ({
  name: 'Harbor Bank',
  sharedAccount: '',
  limit: 50_000_000,
  usage: 6_420_000,
  utilization: 12.84,
  utilizationBand: 'good',
  remainingLimit: 43_580_000,
  cashback: amount(187_000),
  refundYtd: 1_310_000,
  feesPaidYtd: 0,
  netEarnYtd: 1_310_000,
  cards,
});
