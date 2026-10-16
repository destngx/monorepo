import type {
  BillItem,
  BillOccurrence,
  RecurringFrequency,
  RecurringInput,
  RecurringKind,
  RecurringStatus,
  RecurringSuggestion,
  SafeToSpend,
  SuggestionConfidence,
} from '$lib/api/types';
import type { StatusTone } from '$lib/status';

/** Safe-to-spend status: short is serious, nothing left is a warning, else good. */
export function safeToSpendTone(safe: Pick<SafeToSpend, 'shortBy' | 'safeToSpend'>): StatusTone {
  if (safe.shortBy > 0) return 'serious';
  return safe.safeToSpend > 0 ? 'good' : 'warning';
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

export const KIND_LABELS: Record<RecurringKind, string> = {
  income: 'Income',
  bill: 'Bill',
  subscription: 'Subscription',
  insurance: 'Insurance',
  card: 'Card payment',
};

/** An occurrence's status as a badge: always an icon and a label, never colour alone. */
export function billStatus(occurrence: Pick<BillOccurrence, 'status' | 'kind' | 'daysUntil'>): {
  tone: StatusTone;
  label: string;
} {
  const isIncome = occurrence.kind === 'income';
  switch (occurrence.status) {
    case 'paid':
      return { tone: 'good', label: isIncome ? 'Received' : 'Paid' };
    case 'overdue':
      return isIncome ? { tone: 'warning', label: 'Late' } : { tone: 'serious', label: 'Overdue' };
    case 'due':
      if (isIncome) return { tone: 'info', label: 'Expected' };
      return occurrence.daysUntil <= 3 ? { tone: 'warning', label: 'Due soon' } : { tone: 'info', label: 'Due' };
  }
}

/** "Today", "Tomorrow", "In 5 days", "3 days ago". */
export function dueLabel(daysUntil: number): string {
  if (daysUntil === 0) return 'Today';
  if (daysUntil === 1) return 'Tomorrow';
  if (daysUntil === -1) return 'Yesterday';
  return daysUntil > 0 ? `In ${daysUntil} days` : `${-daysUntil} days ago`;
}

export function confidenceStatus(confidence: SuggestionConfidence): { tone: StatusTone; label: string } {
  switch (confidence) {
    case 'high':
      return { tone: 'good', label: 'High confidence' };
    case 'medium':
      return { tone: 'info', label: 'Medium confidence' };
    case 'low':
      return { tone: 'warning', label: 'Low confidence' };
  }
}

/** "Monthly, day 5", "Yearly, 5 Nov", "Quarterly from Feb, day 15", "Statement, pay by the 29th". */
export function scheduleLabel(item: {
  kind: RecurringKind;
  frequency: RecurringFrequency;
  day: number;
  month?: number;
}): string {
  if (item.kind === 'card') return 'Card statement, paid by the 29th (28th in February)';
  const month = item.month ? MONTHS[item.month - 1] : '';
  switch (item.frequency) {
    case 'yearly':
      return `Yearly, ${item.day} ${month}`.trim();
    case 'quarterly':
      return `Quarterly from ${month}, day ${item.day}`;
    default:
      return `Monthly, day ${item.day}`;
  }
}

/** Where a suggestion comes from, in words. */
export function suggestionEvidence(suggestion: RecurringSuggestion): string {
  const spread = suggestion.amountSpread > 0 ? `, amounts within ${suggestion.amountSpread}%` : ', same amount';
  if (suggestion.frequency === 'yearly') {
    return `Seen in ${suggestion.seen} years in a row${spread}`;
  }
  return `Seen in ${suggestion.seen} of the last ${suggestion.checked} months${spread}, within ${suggestion.daySpread} ${
    suggestion.daySpread === 1 ? 'day' : 'days'
  } of day ${suggestion.day}`;
}

/**
 * Calendar rows still needing attention (overdue ones first, then from today on) and the rest of this
 * month so far, so an overdue bill is never tucked away with the settled ones.
 */
export function splitCalendar(
  occurrences: BillOccurrence[],
  asOf: string,
): { earlier: BillOccurrence[]; upcoming: BillOccurrence[] } {
  const isPast = (occurrence: BillOccurrence) => occurrence.expectedDate < asOf;
  return {
    earlier: occurrences.filter((occurrence) => isPast(occurrence) && occurrence.status !== 'overdue'),
    upcoming: [
      ...occurrences.filter((occurrence) => isPast(occurrence) && occurrence.status === 'overdue'),
      ...occurrences.filter((occurrence) => !isPast(occurrence)),
    ],
  };
}

/** Active items first, then paused, each in sheet order. */
export function groupItems(items: BillItem[]): { active: BillItem[]; paused: BillItem[] } {
  return {
    active: items.filter((item) => item.status === 'active'),
    paused: items.filter((item) => item.status === 'paused'),
  };
}

/** The recurring item form; numbers are null while blank. */
export interface RecurringForm {
  name: string;
  kind: RecurringKind;
  amount: number | null;
  frequency: RecurringFrequency;
  day: number | null;
  month: number | null;
  account: string;
  category: string;
  match: string;
  status: RecurringStatus;
  note: string;
}

export function emptyRecurringForm(): RecurringForm {
  return {
    name: '',
    kind: 'bill',
    amount: null,
    frequency: 'monthly',
    day: null,
    month: null,
    account: '',
    category: '',
    match: '',
    status: 'active',
    note: '',
  };
}

export function recurringFormFor(item: BillItem): RecurringForm {
  return {
    name: item.name,
    kind: item.kind,
    amount: item.amount,
    frequency: item.frequency,
    day: item.day || null,
    month: item.month ?? null,
    account: item.account ?? '',
    category: item.category ?? '',
    match: item.match ?? '',
    status: item.status,
    note: item.note ?? '',
  };
}

export function recurringFormFromSuggestion(suggestion: RecurringSuggestion): RecurringForm {
  return {
    name: suggestion.name,
    kind: suggestion.kind,
    amount: suggestion.amount,
    frequency: suggestion.frequency,
    day: suggestion.day,
    month: suggestion.month ?? null,
    account: suggestion.account ?? '',
    category: suggestion.category,
    match: suggestion.match,
    status: 'active',
    note: '',
  };
}

/** Builds the API input, or the first problem to show. The engine validates again. */
export function recurringInputFrom(form: RecurringForm): { input?: RecurringInput; error?: string } {
  const name = form.name.trim();
  if (!name) return { error: 'Name is required.' };
  const isCard = form.kind === 'card';
  if (form.amount !== null && form.amount <= 0) return { error: 'Amount must be more than 0.' };
  if (form.amount === null && !isCard) return { error: 'Amount is required.' };
  if (!isCard && (form.day === null || form.day < 1 || form.day > 31 || !Number.isInteger(form.day)))
    return { error: 'Day must be a day of the month (1-31).' };
  if (form.frequency !== 'monthly' && (form.month === null || form.month < 1 || form.month > 12))
    return { error: 'Pick the month for a quarterly or yearly item.' };
  if (!isCard && !form.match.trim() && !form.category.trim())
    return { error: 'Add match text or a category so payments can be recognised.' };
  return {
    input: {
      name,
      kind: form.kind,
      amount: isCard ? undefined : (form.amount ?? undefined),
      frequency: form.frequency,
      day: isCard ? undefined : (form.day ?? undefined),
      month: form.frequency === 'monthly' ? undefined : (form.month ?? undefined),
      account: form.account.trim() || undefined,
      category: form.category.trim() || undefined,
      match: form.match.trim() || undefined,
      status: form.status,
      note: form.note.trim() || undefined,
    },
  };
}
