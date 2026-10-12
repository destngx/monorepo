import type { BudgetLine, BudgetStatus } from '$lib/api/types';

// WM-BUD-3 status colours and labels.
export const statusTone: Record<BudgetStatus, 'green' | 'amber' | 'red'> = {
  healthy: 'green',
  warning: 'amber',
  urgent: 'red',
};
export const statusLabel: Record<BudgetStatus, string> = { healthy: 'Healthy', warning: 'Warning', urgent: 'Urgent' };
export const statusBar: Record<BudgetStatus, string> = {
  healthy: 'bg-emerald-500',
  warning: 'bg-amber-400',
  urgent: 'bg-rose-500',
};

// Callback shapes shared by the budget components.
export type LineHandler = (line: BudgetLine) => void;
export type CategoryHandler = (category: string) => void;
