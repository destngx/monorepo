import type { CloseItem, CloseItemKey, CloseItemState, MonthlyClose } from '$lib/api/types';
import { formatMonth } from '$lib/format';
import type { StatusTone } from '$lib/status';

/** Where the user acts on each item (paths relative to the app base). */
export function closeLink(key: CloseItemKey, month: string): { href: string; label: string } | null {
  switch (key) {
    case 'drafts':
      return { href: '/transactions/drafts', label: 'Review drafts' };
    case 'categorize':
      return { href: '/transactions', label: 'Open transactions' };
    case 'budget':
      return { href: `/budget?month=${month}`, label: 'Open the budget' };
    case 'goals':
      return { href: '/accounts/goals', label: 'Open goals' };
    case 'moneyLent':
      return { href: '/accounts/lent', label: 'Open money lent' };
    case 'recommendations':
      return { href: '/recommendations', label: 'Open recommendations' };
    case 'snapshot':
      return { href: '/health#net-worth-history', label: 'See the history' };
  }
}

/** Status tone and label per state; the Badge adds the icon, so a state never reads by colour alone. */
export const CLOSE_STATE: Record<CloseItemState, { tone: StatusTone; label: string }> = {
  done: { tone: 'good', label: 'Done' },
  action: { tone: 'warning', label: 'To do' },
  unavailable: { tone: 'neutral', label: 'Unavailable' },
};

/** A figure shown under an item; money is rendered masked by the page. */
export interface CloseFigure {
  label: string;
  value: number;
}

export function closeFigures(item: CloseItem): CloseFigure[] {
  const figures: CloseFigure[] = [];
  const add = (label: string, value: number | null, when = true) => {
    if (when && value !== null) figures.push({ label, value });
  };
  switch (item.key) {
    case 'categorize':
      add('Total', item.amount, (item.count ?? 0) > 0);
      break;
    case 'budget':
      add('Over by', item.amount, (item.count ?? 0) > 0);
      add('Total limit', item.target);
      break;
    case 'goals':
      add('Saved this month', item.amount);
      add('Goals need a month', item.target);
      break;
    case 'moneyLent':
      add('Owed for 30+ days', item.amount, (item.count ?? 0) > 0);
      break;
    case 'recommendations':
      add('Estimated impact a year', item.amount, (item.amount ?? 0) > 0);
      break;
  }
  return figures;
}

/** Headline: how many checks are done, with a tone for the whole close. */
export function closeHeadline(close: MonthlyClose): { text: string; tone: StatusTone } {
  const month = formatMonth(close.month);
  if (close.checked > 0 && close.done === close.checked) {
    return { text: `${month} is closed: all ${close.checked} checks are done.`, tone: 'good' };
  }
  const left = close.checked - close.done;
  return {
    text: `${close.done} of ${close.checked} checks done for ${month}; ${left} ${left === 1 ? 'needs' : 'need'} you.`,
    tone: 'warning',
  };
}

/** The months the page offers: a month and the two before it. */
export function closeMonths(month: string, shift: (month: string, delta: number) => string): string[] {
  return [shift(month, -2), shift(month, -1), month];
}
