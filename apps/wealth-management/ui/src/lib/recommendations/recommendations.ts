import type { LoadFailure, Streamed } from '$lib/api/load';
import type { Recommendation, RecommendationRecord, RecommendationStatus, RecommendationSummary } from '$lib/api/types';
import { formatVND } from '$lib/format';
import type { StatusTone } from '$lib/status';

// The recommendations list (ROADMAP R7): grouping, labels and the headline.

/** Status on the shared status scale; Badge `status` adds the icon, so it never reads by colour alone. */
export const STATUS_TONE: Record<RecommendationStatus, StatusTone> = {
  open: 'warning',
  snoozed: 'info',
  done: 'good',
  dismissed: 'neutral',
};

export const STATUS_LABEL: Record<RecommendationStatus, string> = {
  open: 'Open',
  snoozed: 'Snoozed',
  done: 'Done',
  dismissed: 'Dismissed',
};

export const GROUPS: { status: RecommendationStatus; title: string }[] = [
  { status: 'open', title: 'Open' },
  { status: 'snoozed', title: 'Snoozed' },
  { status: 'done', title: 'Done' },
  { status: 'dismissed', title: 'Dismissed' },
];

const SOURCE_LABELS: Record<string, string> = {
  'account-review': 'Account review',
  'budget-review': 'Budget review',
  'transaction-review': 'Weekly review',
  'credit-summary': 'Credit card summary',
  'year-review': 'Year review',
  'financial-health': 'Health review',
  'intelligence-briefing': 'Briefing',
  'think-tank': 'Think Tank',
  'rule:rebalance': 'Allocation check',
  'rule:goal-plan-short': 'Goal plan check',
  'rule:lent-overdue': 'Money lent check',
  'rule:idle-cash': 'Idle cash check',
  'rule:emergency-fund-rate': 'Emergency fund check',
};

/** Who made the item: a review or an engine check. */
export function sourceLabel(source: string): string {
  if (SOURCE_LABELS[source]) return SOURCE_LABELS[source];
  if (source.startsWith('rule:')) return 'Engine check';
  return source || 'Unknown source';
}

/** Items per status, in the engine's order within each group. */
export function groupByStatus(items: Recommendation[]): Record<RecommendationStatus, Recommendation[]> {
  const groups: Record<RecommendationStatus, Recommendation[]> = { open: [], snoozed: [], done: [], dismissed: [] };
  for (const item of items) groups[item.status]?.push(item);
  return groups;
}

/** "3 open recommendations, worth about 1.200.000 ₫ a year"; render it with MaskedText. */
export function headline(summary: RecommendationSummary): string {
  if (summary.open === 0) return 'No open recommendations';
  const noun = summary.open === 1 ? 'recommendation' : 'recommendations';
  if (summary.openImpactVnd <= 0) return `${summary.open} open ${noun}`;
  return `${summary.open} open ${noun}, worth about ${formatVND(summary.openImpactVnd)} a year`;
}

/** yyyy-mm-dd `days` after `from` (local calendar day). */
export function addDays(from: Date, days: number): string {
  const date = new Date(from.getFullYear(), from.getMonth(), from.getDate() + days);
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${date.getFullYear()}-${month}-${day}`;
}

export const SNOOZE_DAYS = [7, 30] as const;

/** What a review wrote, in a sentence; empty when it wrote nothing. */
export function recordNote(record: RecommendationRecord | undefined): string {
  if (!record) return '';
  if (record.error) return record.error;
  const parts: string[] = [];
  if (record.created.length) parts.push(`${record.created.length} added`);
  if (record.duplicates) parts.push(`${record.duplicates} already on the list`);
  if (record.checked.length)
    parts.push(`${record.checked.length} earlier ${record.checked.length === 1 ? 'item' : 'items'} checked`);
  return parts.length ? `Recommendations: ${parts.join(', ')}.` : 'No new recommendations from this review.';
}

/**
 * The list to show first: the stored read, or (when the read fails) the list
 * the rule refresh returns, so either source fills the page. Never rejects.
 */
export async function storedOrChecked<T>(stored: Streamed<T>, checked: Streamed<T>): Streamed<T> {
  const first = await stored;
  if (first.ok) return first;
  const second = await checked;
  return second.ok ? second : first;
}

/**
 * The rule refresh writes the Recommendations tab, so a connection without
 * write access is refused (401/403): the checks did not run and the stored
 * list stands. Google Sheets credential problems are not this case; the
 * layout's re-auth alert explains those.
 */
export function checksSkipped(failure: LoadFailure | null): boolean {
  return !!failure && !failure.sheetsAuth && (failure.status === 401 || failure.status === 403);
}
