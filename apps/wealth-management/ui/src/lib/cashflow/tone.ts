import type { IconName } from '$lib/components/Icon.svelte';
import type { VerdictTone } from './yearly';

// Status colour always comes with an icon and a title (never colour alone).
export const TONE_STYLE: Record<VerdictTone, { icon: IconName; box: string; ink: string }> = {
  good: { icon: 'check', box: 'border-l-emerald-500', ink: 'text-emerald-700 dark:text-emerald-300' },
  warn: { icon: 'alert', box: 'border-l-amber-500', ink: 'text-amber-700 dark:text-amber-300' },
  bad: { icon: 'alert', box: 'border-l-rose-500', ink: 'text-rose-700 dark:text-rose-300' },
  neutral: { icon: 'info', box: 'border-l-slate-400', ink: '' },
};
