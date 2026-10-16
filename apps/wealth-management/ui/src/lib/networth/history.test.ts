import { describe, expect, it } from 'vitest';
import type { NetWorthMilestone } from '$lib/api/types';
import { shiftMonth } from '$lib/format';
import {
  estimatedCount,
  lineSegments,
  milestonesInRange,
  milestoneText,
  pointsInRange,
  rangeChange,
  sourceLabel,
} from './history';
import { point } from './history.testdata';

const months = (count: number) =>
  Array.from({ length: count }, (_, i) =>
    point(shiftMonth('2025-01', i), 100_000_000 + i * 1_000_000, i < 3 ? 'estimated' : 'snapshot'),
  );

describe('net worth history', () => {
  it('keeps the last 13 months for 1Y and every month for All', () => {
    const points = months(20);
    expect(pointsInRange(points, '1Y')).toHaveLength(13);
    expect(pointsInRange(points, '1Y')[0].month).toBe('2025-08');
    expect(pointsInRange(points, 'All')).toHaveLength(20);
  });

  it('measures the change over the range and flags an estimated start', () => {
    const points = months(5);
    expect(rangeChange(points)).toMatchObject({ change: 4_000_000, percent: 4, estimated: true });
    expect(rangeChange(points.slice(3))?.estimated).toBe(false);
    expect(rangeChange(points.slice(0, 1))).toBeNull();
    expect(rangeChange([point('2026-01', 0), point('2026-02', 5)])?.percent).toBeNull();
  });

  it('dashes the segments that touch an estimated month', () => {
    expect(lineSegments(months(5)).map((s) => s.dashed)).toEqual([true, true, true, false]);
  });

  it('names sources in words', () => {
    expect(sourceLabel(point('2026-01', 1, 'live'))).toBe('Today');
    expect(sourceLabel(point('2026-01', 1, 'estimated'))).toBe('Estimated');
    expect(sourceLabel(point('2026-01', 1))).toBe('Snapshot');
    expect(estimatedCount(months(5))).toBe(3);
  });

  it('keeps the milestones inside the range', () => {
    const milestones: NetWorthMilestone[] = [
      {
        month: '2025-02',
        kind: 'threshold',
        threshold: 1e8,
        netWorth: 1e8,
        label: 'Crossed 100 million VND',
        estimated: true,
      },
      {
        month: '2025-05',
        kind: 'all_time_high',
        threshold: null,
        netWorth: 2e8,
        label: 'All-time high',
        estimated: false,
      },
    ];
    expect(milestonesInRange(milestones, months(5).slice(3))).toHaveLength(1);
    expect(milestoneText(milestones[1])).toBe('All-time high');
  });
});
