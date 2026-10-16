import type { GoalPoolSettings, PreferenceValue } from '$lib/api/types';

// Goal pool settings form (WM-SET-8). Ranges match the engine's validation.

type GoalPoolKey = keyof GoalPoolSettings;

/** Inputs hold text while typing; an empty field resets the key to its default. */
export type GoalPoolDraft = Record<GoalPoolKey, string | number>;

export const GOAL_POOL_FIELDS: {
  key: GoalPoolKey;
  preference: PreferenceValue['key'];
  label: string;
  hint: string;
  min: number;
  max: number;
  fallback: number;
}[] = [
  {
    key: 'emergencyMonths',
    preference: 'goal.emergencyMonths',
    label: 'Emergency fund',
    hint: 'Target in months of needs (fixed and essential spending).',
    min: 1,
    max: 24,
    fallback: 6,
  },
  {
    key: 'spendingBufferMonths',
    preference: 'goal.spendingBufferMonths',
    label: 'Spending buffer',
    hint: 'Months of spending kept out of the goal pool.',
    min: 0,
    max: 12,
    fallback: 1,
  },
];

function parse(value: string | number): number | null {
  if (typeof value === 'number') return Number.isFinite(value) ? value : null;
  const trimmed = value.trim();
  return trimmed === '' ? null : Number(trimmed);
}

/** Validation message for the form, or null when it can be saved. */
export function goalPoolError(draft: GoalPoolDraft): string | null {
  for (const field of GOAL_POOL_FIELDS) {
    const value = parse(draft[field.key]);
    if (value === null) continue;
    if (!Number.isFinite(value) || value < field.min || value > field.max) {
      return `${field.label} must be between ${field.min} and ${field.max} months.`;
    }
  }
  return null;
}

/** The keys that differ from the saved settings; an empty field clears the key. */
export function goalPoolChanges(saved: GoalPoolSettings, draft: GoalPoolDraft): PreferenceValue[] {
  const changes: PreferenceValue[] = [];
  for (const field of GOAL_POOL_FIELDS) {
    const value = parse(draft[field.key]);
    if ((value ?? field.fallback) !== saved[field.key]) {
      changes.push({ key: field.preference, value });
    }
  }
  return changes;
}
