// Display formatting. Base currency is VND with vi-VN grouping and no
// decimals; dates display as dd/MM/yyyy (WM-UI-5).

const vnd = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 });
const compact = new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 });
const usd = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export const MASK = '••••••';

export function formatVND(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  // Avoid "-0 ₫" after rounding.
  const rounded = Math.round(value) || 0;
  return `${vnd.format(rounded)} ₫`;
}

/** Short axis labels: 12.5M, 300K. */
export function formatCompact(value: number): string {
  return compact.format(value);
}

export function formatMoney(value: number | null | undefined, currency = 'VND'): string {
  if (currency === 'VND') return formatVND(value);
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  return `${usd.format(value)} ${currency}`;
}

export function formatPercent(value: number | null | undefined, digits = 1): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  return `${value.toFixed(digits)}%`;
}

export function formatSignedPercent(value: number | null | undefined, digits = 1): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  return `${value > 0 ? '+' : ''}${value.toFixed(digits)}%`;
}

/**
 * Formats an ISO date or timestamp as dd/MM/yyyy. Only the calendar part is
 * used, so a sheet date is never shifted by the browser's timezone.
 */
export function formatDate(value: string | null | undefined): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value ?? '');
  if (!match) return value ?? '-';
  return `${match[3]}/${match[2]}/${match[1]}`;
}

/** "2026-10" → "Oct 2026"; with short, "Oct". */
export function formatMonth(month: string, short = false): string {
  const match = /^(\d{4})-(\d{2})/.exec(month);
  if (!match) return month;
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 1));
  return date.toLocaleString('en', { month: 'short', year: short ? undefined : 'numeric', timeZone: 'UTC' });
}

/** Today's date in Asia/Ho_Chi_Minh as yyyy-MM-dd. */
export function todayISO(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(now);
}

export function currentMonth(now = new Date()): string {
  return todayISO(now).slice(0, 7);
}

export function shiftMonth(month: string, delta: number): string {
  const [year, monthIndex] = month.split('-').map(Number);
  const date = new Date(Date.UTC(year, monthIndex - 1 + delta, 1));
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
}

/** Signed amount of a transaction: deposit minus payment. */
export function transactionAmount(tx: { payment: number | null; deposit: number | null }): number {
  return (tx.deposit ?? 0) - (tx.payment ?? 0);
}
