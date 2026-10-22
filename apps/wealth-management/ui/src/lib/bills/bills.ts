import type {
  AccountMinimumBalance,
  BillItem,
  ExclusionReason,
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

/**
 * Safe-to-spend status (D36): short (committed money not covered) is serious; tight (essentials not covered)
 * and nothing free are warnings; free money is good. Always a label, never colour alone.
 */
export function safeToSpendBadge(safe: Pick<SafeToSpend, 'shortBy' | 'tightBy' | 'safeToSpend'>): {
  tone: StatusTone;
  label: string;
} {
  if (safe.shortBy > 0) return { tone: 'serious', label: 'Short' };
  if (safe.tightBy > 0) return { tone: 'warning', label: 'Tight' };
  return safe.safeToSpend > 0 ? { tone: 'good', label: 'Covered' } : { tone: 'warning', label: 'Nothing free' };
}

/** A minimum balance's status: always an icon and a label, never colour alone (D35). */
export function minimumStatus(track: Pick<AccountMinimumBalance, 'status' | 'daysLeft'>): {
  tone: StatusTone;
  label: string;
} {
  if (track.status === 'on_track') return { tone: 'good', label: 'On track' };
  return { tone: track.daysLeft <= 7 ? 'serious' : 'warning', label: 'At risk' };
}

/** A row of the minimum balance editor: an account that can keep a minimum, and its tracking if set. */
export interface MinimumRow {
  account: string;
  track: AccountMinimumBalance | null;
}

/**
 * The accounts that can keep a minimum (ready and not-ready ones; never crypto), with the
 * tracking of those that have one. Accounts with a minimum come first, in the setting's order.
 */
export function minimumRows(safe: SafeToSpend): MinimumRow[] {
  const rows: MinimumRow[] = safe.minimumBalances.map((track) => ({ account: track.account, track }));
  const known = new Set(rows.map((row) => row.account.trim().toLowerCase()));
  const candidates = [
    ...safe.readyAccounts.map((account) => account.name),
    ...safe.excludedAccounts.filter((account) => account.reason === 'not_ready').map((account) => account.name),
  ];
  for (const name of candidates) {
    const key = name.trim().toLowerCase();
    if (known.has(key)) continue;
    known.add(key);
    rows.push({ account: name, track: null });
  }
  return rows;
}

/** Why an account is left out of ready money, as shown next to it. */
export const EXCLUSION_LABELS: Record<ExclusionReason, string> = {
  crypto: 'crypto, not cash you can spend today',
  not_ready: 'marked not ready to use (slow to withdraw)',
};

/** The most grace days an item can have (the engine's MaxGraceDays). */
export const MAX_GRACE_DAYS = 21;

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
      // Past its date but within its grace days: late, not urgent.
      if (occurrence.daysUntil < 0) return { tone: 'info', label: 'Late, in grace' };
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

/** "Monthly, day 5", "Yearly, 5 Nov", "Quarterly from Feb, day 15", "Statement, pay by the 29th", "…, 14 days grace". */
export function scheduleLabel(item: {
  kind: RecurringKind;
  frequency: RecurringFrequency;
  day: number;
  month?: number;
  graceDays?: number;
}): string {
  if (item.kind === 'card') return 'Card statement, paid by the 29th (28th in February)';
  const grace = item.graceDays ? `, ${item.graceDays} ${item.graceDays === 1 ? 'day' : 'days'} grace` : '';
  const month = item.month ? MONTHS[item.month - 1] : '';
  switch (item.frequency) {
    case 'yearly':
      return `Yearly, ${item.day} ${month}`.trim() + grace;
    case 'quarterly':
      return `Quarterly from ${month}, day ${item.day}${grace}`;
    default:
      return `Monthly, day ${item.day}${grace}`;
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
 * Calendar rows still needing attention (overdue ones first, then late ones within their grace, then
 * from today on) and the rest of this month so far, so an unpaid bill is never tucked away with the
 * settled ones.
 */
export function splitCalendar(
  occurrences: BillOccurrence[],
  asOf: string,
): { earlier: BillOccurrence[]; upcoming: BillOccurrence[] } {
  const isPast = (occurrence: BillOccurrence) => occurrence.expectedDate < asOf;
  return {
    earlier: occurrences.filter((occurrence) => isPast(occurrence) && occurrence.status === 'paid'),
    upcoming: [
      ...occurrences.filter((occurrence) => isPast(occurrence) && occurrence.status === 'overdue'),
      ...occurrences.filter((occurrence) => isPast(occurrence) && occurrence.status === 'due'),
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
  graceDays: number | null;
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
    graceDays: null,
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
    graceDays: item.graceDays || null,
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
    graceDays: null,
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
  if (
    !isCard &&
    form.graceDays !== null &&
    (form.graceDays < 0 || form.graceDays > MAX_GRACE_DAYS || !Number.isInteger(form.graceDays))
  )
    return { error: `Grace days must be a whole number from 0 to ${MAX_GRACE_DAYS}.` };
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
      graceDays: isCard ? undefined : form.graceDays || undefined,
    },
  };
}
