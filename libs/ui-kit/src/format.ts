/** What a missing or unreadable value shows. */
export const MISSING = '-';

/** The engines' calendar: "today" and dates without a zone are in this time zone. */
export const APP_TIME_ZONE = 'Asia/Ho_Chi_Minh';

/**
 * A calendar date (yyyy-MM-dd, optionally with a time after it) as dd/MM/yyyy.
 * It reads only the date part, so no time zone can shift the day. Text that
 * is not a date ("Pending") is shown as it is; nothing shows MISSING.
 */
export function formatDate(value: string | null | undefined): string {
  if (!value) return MISSING;
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  return match ? `${match[3]}/${match[2]}/${match[1]}` : value;
}

/** An instant as dd/MM/yyyy HH:mm in the browser's time zone (or `timeZone`); MISSING when unreadable. */
export function formatDateTime(value: string | null | undefined, timeZone?: string): string {
  const date = new Date(value ?? '');
  if (!value || Number.isNaN(date.getTime())) return MISSING;
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat('en-GB', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    })
      .formatToParts(date)
      .map((part) => [part.type, part.value]),
  );
  return `${parts.day}/${parts.month}/${parts.year} ${parts.hour}:${parts.minute}`;
}

/** Today's date in APP_TIME_ZONE as yyyy-MM-dd. */
export function todayISO(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: APP_TIME_ZONE }).format(now);
}

/** A file size in B, KB or MB. */
export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return MISSING;
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
