import type { NeedsYou, NeedsYouItem, NeedsYouKind, NeedsYouStatus } from '$lib/api/types';
import type { IconName } from '$lib/components/Icon.svelte';
import type { StatusTone } from '$lib/status';

// The home inbox (ROADMAP R5): the engine ranks the items; these helpers
// word the headline and pick what each row shows.

/** How many rows show before "Show all". */
export const NEEDS_YOU_VISIBLE = 6;

export const KIND_ICON: Record<NeedsYouKind, IconName> = {
  drafts: 'inbox',
  'bill-overdue': 'calendar',
  'card-due': 'card',
  'short-before-payday': 'wallet',
  'tight-before-payday': 'wallet',
  'minimum-balance': 'wallet',
  'budget-over-pace': 'budget',
  'lent-overdue': 'loan',
  'goal-plan-short': 'goal',
  'goal-behind': 'goal',
  rebalance: 'layers',
  uncategorized: 'transactions',
  recommendation: 'lightbulb',
  'idle-cash': 'investments',
  'emergency-fund-rate': 'investments',
  'wallet-switch': 'investments',
  'month-close': 'check',
};

/** What the row's amount means; per month or per year where it is a rate. */
export function itemAmount(item: NeedsYouItem): { value: number; unit: string } | null {
  switch (item.kind) {
    case 'goal-plan-short':
    case 'goal-behind':
      return item.amountVnd === null ? null : { value: item.amountVnd, unit: 'a month' };
    case 'idle-cash':
    case 'emergency-fund-rate':
    case 'wallet-switch':
    case 'recommendation':
      return item.impactVnd === null ? null : { value: item.impactVnd, unit: 'a year' };
    default:
      return item.amountVnd === null ? null : { value: item.amountVnd, unit: '' };
  }
}

const SEVERITY: NeedsYouStatus[] = ['critical', 'serious', 'warning', 'info'];

const STATUS_WORD: Record<NeedsYouStatus, [string, string]> = {
  critical: ['critical', 'critical'],
  serious: ['serious', 'serious'],
  warning: ['to watch', 'to watch'],
  info: ['suggestion', 'suggestions'],
};

export interface InboxHeadline {
  tone: StatusTone;
  title: string;
  text: string;
}

/** The answer to "What needs me?": the count and the most severe status. */
export function inboxHeadline(inbox: NeedsYou): InboxHeadline {
  const missing = inbox.unavailable.map((part) => part.label.toLowerCase());
  if (!inbox.items.length) {
    if (missing.length) {
      return {
        tone: 'neutral',
        title: 'Nothing needs you in what could be checked',
        text: `Could not check ${joinWords(missing)}; try again later.`,
      };
    }
    return {
      tone: 'good',
      title: 'Nothing needs you',
      text: 'No bills overdue or due soon, the budget is on pace and there are no drafts or open suggestions.',
    };
  }
  const counts = SEVERITY.map(
    (status) => [status, inbox.items.filter((item) => item.status === status).length] as const,
  );
  const worst = counts.find(([, count]) => count > 0)?.[0] ?? 'info';
  const total = inbox.items.length;
  const parts = counts
    .filter(([, count]) => count > 0)
    .map(([status, count]) => `${count} ${STATUS_WORD[status][count === 1 ? 0 : 1]}`);
  let text = `${joinWords(parts)}, most urgent first.`;
  if (missing.length) text += ` Could not check ${joinWords(missing)}.`;
  return { tone: worst, title: total === 1 ? '1 thing needs you' : `${total} things need you`, text };
}

function joinWords(words: string[]): string {
  if (words.length <= 1) return words.join('');
  return `${words.slice(0, -1).join(', ')} and ${words[words.length - 1]}`;
}
