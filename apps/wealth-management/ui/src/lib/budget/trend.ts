import type { SpendPattern, SpendTrend } from '$lib/api/types';

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
