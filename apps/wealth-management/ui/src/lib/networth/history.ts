import type { NetWorthHistory, NetWorthMilestone, NetWorthPoint } from '$lib/api/types';

// Net worth history (ROADMAP P3): 1Y and all-time views of the engine's monthly points.

export const NET_WORTH_RANGES = ['1Y', 'All'] as const;
export type NetWorthRange = (typeof NET_WORTH_RANGES)[number];

/** 1Y is the last 13 months, so the first point is a year before the last. */
export function pointsInRange(points: NetWorthPoint[], range: NetWorthRange): NetWorthPoint[] {
  return range === 'All' ? points : points.slice(-13);
}

export function milestonesInRange(milestones: NetWorthMilestone[], points: NetWorthPoint[]): NetWorthMilestone[] {
  const months = new Set(points.map((point) => point.month));
  return milestones.filter((milestone) => months.has(milestone.month));
}

/** Change from the first to the last point of the range; estimated when either end is. */
export function rangeChange(
  points: NetWorthPoint[],
): { change: number; percent: number | null; from: NetWorthPoint; estimated: boolean } | null {
  if (points.length < 2) return null;
  const from = points[0];
  const to = points[points.length - 1];
  const change = to.netWorth - from.netWorth;
  return {
    change,
    percent: from.netWorth === 0 ? null : (change / Math.abs(from.netWorth)) * 100,
    from,
    estimated: from.estimated || to.estimated,
  };
}

/** Plain words for where a point comes from; shown with the marker shape, never by colour alone. */
export function sourceLabel(point: NetWorthPoint): string {
  switch (point.source) {
    case 'snapshot':
      return 'Snapshot';
    case 'live':
      return 'Today';
    default:
      return 'Estimated';
  }
}

/** Line segments between neighbours: dashed when either end is estimated. */
export function lineSegments(points: NetWorthPoint[]): { from: number; to: number; dashed: boolean }[] {
  return points.slice(1).map((point, index) => ({
    from: index,
    to: index + 1,
    dashed: point.estimated || points[index].estimated,
  }));
}

export function milestoneText(milestone: NetWorthMilestone): string {
  return milestone.kind === 'all_time_high' ? 'All-time high' : 'Crossed';
}

/** How many points of the range are estimated, for the note under the chart. */
export function estimatedCount(points: NetWorthPoint[]): number {
  return points.filter((point) => point.estimated).length;
}

export function hasHistory(history: NetWorthHistory): boolean {
  return history.points.length > 1;
}
