import type {
  Allocation,
  AllocationLine,
  AllocationStatus,
  AllocationTargets,
  PreferenceValue,
  TargetClass,
} from '$lib/api/types';
import { PALETTE } from '$lib/charts/palette';
import type { StatusTone } from '$lib/status';

// The engine computes the one allocation (ROADMAP C4, WM-INV-20); this
// module only presents it and edits the targets kept in the Preferences tab.

export const TARGET_CLASSES: { key: TargetClass; label: string }[] = [
  { key: 'cash', label: 'Cash and deposits' },
  { key: 'funds', label: 'Fund certificates' },
  { key: 'crypto', label: 'Crypto' },
];

const TARGET_KEYS: Record<TargetClass, PreferenceValue['key']> = {
  cash: 'allocation.target.cash',
  funds: 'allocation.target.funds',
  crypto: 'allocation.target.crypto',
};

/** One fixed colour per class, from the validated series palette, so every page draws a class alike. */
export const ALLOCATION_COLORS: Record<AllocationLine['class'], string> = {
  cash: PALETTE[0],
  funds: PALETTE[1],
  crypto: PALETTE[2],
  moneyLent: PALETTE[8],
};

/** How a line reads against its target; always shown with the status icon and this label. */
export function lineStatus(status: AllocationStatus): { tone: StatusTone; label: string } {
  switch (status) {
    case 'over':
      return { tone: 'warning', label: 'Over target' };
    case 'under':
      return { tone: 'warning', label: 'Under target' };
    case 'on_target':
      return { tone: 'good', label: 'On target' };
    default:
      return { tone: 'neutral', label: 'No target' };
  }
}

/** The headline status of the whole allocation. */
export function allocationStatus(allocation: Allocation): { tone: StatusTone; label: string } {
  if (!allocation.targets) return { tone: 'neutral', label: 'No targets set' };
  return allocation.rebalance
    ? { tone: 'warning', label: 'Rebalance suggested' }
    : { tone: 'good', label: 'On target' };
}

/** "Cash and deposits is over target by 10.0 points" for each class beyond the threshold. */
export function breachSummary(allocation: Allocation): string[] {
  return allocation.lines
    .filter((line) => (line.status === 'over' || line.status === 'under') && line.drift !== null)
    .map(
      (line) =>
        `${line.label} is ${line.status === 'over' ? 'over' : 'under'} target by ${Math.abs(line.drift ?? 0).toFixed(1)} points`,
    );
}

/** Donut slices of the lines with a value, in class order. */
export function allocationSlices(allocation: Allocation): { label: string; value: number; color: string }[] {
  return allocation.lines
    .filter((line) => line.value > 0)
    .map((line) => ({ label: line.label, value: line.value, color: ALLOCATION_COLORS[line.class] }));
}

/** Validation message for a target set, or null when the engine will accept it. */
export function targetsError(targets: AllocationTargets): string | null {
  for (const { key, label } of TARGET_CLASSES) {
    const value = targets[key];
    if (!Number.isFinite(value) || value < 0 || value > 100) return `${label} must be between 0 and 100%.`;
  }
  const sum = TARGET_CLASSES.reduce((total, { key }) => total + targets[key], 0);
  if (Math.abs(sum - 100) > 0.01) return `Targets must add up to 100% (now ${Number(sum.toFixed(2))}%).`;
  return null;
}

/** Preference values saving the three targets together, or clearing them with null. */
export function targetValues(targets: AllocationTargets | null): PreferenceValue[] {
  return TARGET_CLASSES.map(({ key }) => ({ key: TARGET_KEYS[key], value: targets ? targets[key] : null }));
}

/**
 * Targets an older version kept in this browser (JSON), read once to offer
 * moving them to the sheet; null when nothing valid is stored.
 */
export function parseBrowserTargets(raw: string): AllocationTargets | null {
  if (!raw) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // A corrupt stored value is treated as "no targets": there is nothing to move.
    return null;
  }
  if (!parsed || typeof parsed !== 'object') return null;
  const record = parsed as Record<string, unknown>;
  const targets = {} as AllocationTargets;
  for (const { key } of TARGET_CLASSES) {
    const value = record[key];
    if (typeof value !== 'number') return null;
    targets[key] = value;
  }
  return targetsError(targets) ? null : targets;
}
