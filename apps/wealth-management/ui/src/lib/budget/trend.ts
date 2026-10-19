import type { CategoryNature, LimitDirection, SpendPattern, SpendTrend } from '$lib/api/types';

const TRENDS: Record<SpendTrend, string> = {
  rising: 'Rising',
  falling: 'Falling',
  steady: 'Steady',
  unknown: '',
};

const PATTERNS: Record<SpendPattern, string> = {
  steady: '',
  variable: 'swings month to month',
  occasional: 'only some months',
};

/** Short words for a category's spend trend and pattern (WM-BUD-9), e.g. "Rising, swings month to month". */
export function spendShape(trend: SpendTrend, pattern: SpendPattern): string {
  const words = [TRENDS[trend], PATTERNS[pattern]].filter(Boolean).join(', ');
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/** Label and badge hue for a limit move (WM-BUD-10); the label carries the meaning, never the hue alone. */
export const DIRECTIONS: Record<LimitDirection, { label: string; tone: 'blue' | 'violet' | 'neutral' }> = {
  increase: { label: 'Increase', tone: 'blue' },
  lower: { label: 'Lower', tone: 'violet' },
  keep: { label: 'Keep', tone: 'neutral' },
};

const NATURES: Partial<Record<CategoryNature, string>> = {
  fixed: 'Fixed cost',
  essential: 'Need',
  flexible: 'Want',
};

/** What a category is for the quality-of-life rules: "Fixed cost", "Need", "Want", or "Untagged". */
export function natureLabel(nature: CategoryNature | undefined): string {
  return (nature && NATURES[nature]) || 'Untagged';
}
