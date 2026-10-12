import { todayISO } from '$lib/format';

// Daily entry reminder (WM-TX-9): an opt-in banner, shown on app load after a
// chosen hour when nothing is dated today. No OS notifications.

export const DEFAULT_REMINDER_HOUR = 20;

/** Hour of day (0-23) in Asia/Ho_Chi_Minh, the ledger's timezone. */
export function ledgerHour(now: Date): number {
  const hour = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', hourCycle: 'h23', timeZone: 'Asia/Ho_Chi_Minh' })
    .format(now)
    .slice(0, 2);
  return Number(hour);
}

/** A stored hour setting as 0-23, falling back to the default. */
export function parseReminderHour(value: string): number {
  const hour = Number(value);
  return value.trim() !== '' && Number.isInteger(hour) && hour >= 0 && hour <= 23 ? hour : DEFAULT_REMINDER_HOUR;
}

export interface ReminderCheck {
  enabled: boolean;
  hour: number;
  now: Date;
  /** The day ("YYYY-MM-DD") the banner was last dismissed, if any. */
  dismissedOn: string;
}

/** Whether to look at today's transactions at all; saves a fetch when not. */
export function reminderDue({ enabled, hour, now, dismissedOn }: ReminderCheck): boolean {
  return enabled && dismissedOn !== todayISO(now) && ledgerHour(now) >= hour;
}

export function hasEntryToday(transactions: { date: string }[], now: Date): boolean {
  const today = todayISO(now);
  return transactions.some((tx) => tx.date.startsWith(today));
}
