import type { BudgetLine, BudgetStatus } from '$lib/api/types';
import { STATUS, type StatusTone } from '$lib/status';

// WM-BUD-3 statuses on the shared status scale, with their own labels.
export const statusTone: Record<BudgetStatus, StatusTone> = {
  healthy: 'good',
  warning: 'warning',
  urgent: 'critical',
};
export const statusLabel: Record<BudgetStatus, string> = { healthy: 'Healthy', warning: 'Warning', urgent: 'Urgent' };
export const statusBar: Record<BudgetStatus, string> = {
  healthy: STATUS.good.fill,
  warning: STATUS.warning.fill,
  urgent: STATUS.critical.fill,
};

// Callback shapes shared by the budget components.
export type LineHandler = (line: BudgetLine) => void;
export type CategoryHandler = (category: string) => void;
