import { type IconName } from '@libs/ui-kit';
import type { InsightIcon, InsightSeverity } from '$lib/api/types';
import { STATUS, type StatusToken, type StatusTone } from '$lib/status';

/** AI severities on the shared status scale. */
export const severityTone: Record<InsightSeverity, StatusTone> = {
  info: 'info',
  warning: 'warning',
  success: 'good',
  critical: 'critical',
};

export function severityStatus(severity: InsightSeverity): StatusToken {
  return STATUS[severityTone[severity]] ?? STATUS.info;
}

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
