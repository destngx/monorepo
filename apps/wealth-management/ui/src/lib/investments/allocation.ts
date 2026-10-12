import type { PortfolioTotals } from '$lib/api/types';

// Target allocation and drift (WM-INV-20). Classes follow the engine
// valuation totals (WM-RULE-8): VND accounts, fund certificates, crypto.

export type AssetClass = 'cash' | 'funds' | 'crypto';
export type Targets = Record<AssetClass, number>;

export const ASSET_CLASSES: { key: AssetClass; label: string }[] = [
  { key: 'cash', label: 'Cash (VND accounts)' },
  { key: 'funds', label: 'Fund certificates' },
  { key: 'crypto', label: 'Crypto' },
];

/** Rebalance when any class is more than this many percentage points off target. */
export const DRIFT_THRESHOLD = 5;

/** Value per class from the engine totals. */
export function classValues(totals: Pick<PortfolioTotals, 'vndAccounts' | 'funds' | 'crypto'>): Targets {
  return { cash: totals.vndAccounts, funds: totals.funds, crypto: totals.crypto };
}

/** Targets stored as JSON; null when nothing (or nothing valid) is stored. */
export function parseTargets(raw: string): Targets | null {
  if (!raw) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // A corrupt stored value means "no targets set"; the user sets them again.
    return null;
  }
  if (!parsed || typeof parsed !== 'object') return null;
  const record = parsed as Record<string, unknown>;
  const targets = {} as Targets;
  for (const { key } of ASSET_CLASSES) {
    const value = record[key];
    if (typeof value !== 'number' || !Number.isFinite(value) || value < 0 || value > 100) return null;
    targets[key] = value;
  }
  return targets;
}

export function serializeTargets(targets: Targets): string {
  return JSON.stringify(targets);
}

/** Validation message for a target set, or null when it is usable. */
export function targetsError(targets: Targets): string | null {
  for (const { key, label } of ASSET_CLASSES) {
    const value = targets[key];
    if (!Number.isFinite(value) || value < 0 || value > 100) return `${label} must be between 0 and 100%.`;
  }
  const sum = ASSET_CLASSES.reduce((total, { key }) => total + targets[key], 0);
  if (Math.abs(sum - 100) > 0.01) return `Targets must add up to 100% (now ${Number(sum.toFixed(2))}%).`;
  return null;
}

export interface DriftRow {
  key: AssetClass;
  label: string;
  value: number;
  currentPercent: number;
  targetPercent: number;
  /** Current minus target, in percentage points. */
  drift: number;
  breach: boolean;
}

/** Drift per class against the targets; `shouldRebalance` when any class is beyond the threshold. */
export function computeDrift(
  values: Targets,
  targets: Targets,
  threshold = DRIFT_THRESHOLD,
): { rows: DriftRow[]; total: number; shouldRebalance: boolean } {
  const total = ASSET_CLASSES.reduce((sum, { key }) => sum + Math.max(values[key], 0), 0);
  const rows = ASSET_CLASSES.map(({ key, label }) => {
    const value = Math.max(values[key], 0);
    const currentPercent = total ? (value / total) * 100 : 0;
    const drift = currentPercent - targets[key];
    return {
      key,
      label,
      value,
      currentPercent,
      targetPercent: targets[key],
      drift,
      breach: Math.abs(drift) > threshold,
    };
  });
  return { rows, total, shouldRebalance: total > 0 && rows.some((row) => row.breach) };
}
