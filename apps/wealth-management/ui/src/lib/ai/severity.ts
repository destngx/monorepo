import type { IconName } from '$lib/components/Icon.svelte';
import type { InsightIcon, InsightSeverity } from '$lib/api/types';

export const severityStyle: Record<InsightSeverity, string> = {
  info: 'border-sky-500/30 bg-sky-500/5 text-sky-700 dark:text-sky-300',
  warning: 'border-amber-500/30 bg-amber-500/5 text-amber-700 dark:text-amber-300',
  success: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-300',
  critical: 'border-rose-500/30 bg-rose-500/5 text-rose-700 dark:text-rose-300',
};

export const insightIcon: Record<InsightIcon, IconName> = {
  alert: 'alert',
  'trend-up': 'investments',
  'trend-down': 'trendDown',
  savings: 'loan',
  tip: 'sparkles',
  target: 'goal',
  shield: 'shield',
  chart: 'budget',
  wallet: 'card',
  clock: 'clock',
  zap: 'zap',
  star: 'star',
};
