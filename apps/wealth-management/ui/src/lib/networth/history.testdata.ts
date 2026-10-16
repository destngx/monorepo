import type { NetWorthHistory, NetWorthPoint } from '$lib/api/types';

// Invented figures for the net worth history tests.

export const point = (
  month: string,
  netWorth: number,
  source: NetWorthPoint['source'] = 'snapshot',
): NetWorthPoint => ({
  month,
  asOf: `${month}-28`,
  netWorth,
  source,
  estimated: source === 'estimated',
  change: null,
  savings: null,
  market: null,
});

/** Aug 2026 to Nov 2026: an estimated month, two snapshots and today. */
export function netWorthHistory(overrides: Partial<NetWorthHistory> = {}): NetWorthHistory {
  const points: NetWorthPoint[] = [
    point('2026-08', 90_000_000, 'estimated'),
    { ...point('2026-09', 95_000_000), change: 5_000_000 },
    { ...point('2026-10', 110_000_000), change: 15_000_000, savings: 12_000_000, market: 3_000_000 },
    { ...point('2026-11', 108_000_000, 'live'), change: -2_000_000, savings: 1_000_000, market: -3_000_000 },
  ];
  return {
    asOf: '2026-11-05',
    current: 108_000_000,
    points,
    milestones: [
      {
        month: '2026-10',
        kind: 'threshold',
        threshold: 100_000_000,
        netWorth: 110_000_000,
        label: 'Crossed 100 million VND',
        estimated: false,
      },
      {
        month: '2026-10',
        kind: 'all_time_high',
        threshold: null,
        netWorth: 110_000_000,
        label: 'All-time high',
        estimated: false,
      },
    ],
    snapshots: 2,
    estimated: 1,
    tabExists: true,
    issues: [],
    ...overrides,
  };
}
