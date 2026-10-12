import { describe, expect, it } from 'vitest';
import { hasEntryToday, ledgerHour, parseReminderHour, reminderDue } from './reminder';

// 2026-10-13 20:30 in Asia/Ho_Chi_Minh (UTC+7).
const evening = new Date('2026-10-13T13:30:00Z');
const morning = new Date('2026-10-13T01:00:00Z');

describe('entry reminder', () => {
  it('reads the hour in the ledger timezone', () => {
    expect(ledgerHour(evening)).toBe(20);
    expect(ledgerHour(morning)).toBe(8);
    expect(ledgerHour(new Date('2026-10-13T17:00:00Z'))).toBe(0);
  });

  it('parses the hour setting with a default', () => {
    expect(parseReminderHour('21')).toBe(21);
    expect(parseReminderHour('0')).toBe(0);
    expect(parseReminderHour('')).toBe(20);
    expect(parseReminderHour('25')).toBe(20);
    expect(parseReminderHour('abc')).toBe(20);
  });

  it('is due only when enabled, after the hour and not dismissed today', () => {
    const base = { enabled: true, hour: 20, now: evening, dismissedOn: '' };
    expect(reminderDue(base)).toBe(true);
    expect(reminderDue({ ...base, enabled: false })).toBe(false);
    expect(reminderDue({ ...base, now: morning })).toBe(false);
    expect(reminderDue({ ...base, dismissedOn: '2026-10-13' })).toBe(false);
    expect(reminderDue({ ...base, dismissedOn: '2026-10-12' })).toBe(true);
  });

  it('looks for a transaction dated today in the ledger timezone', () => {
    expect(hasEntryToday([{ date: '2026-10-12' }], evening)).toBe(false);
    expect(hasEntryToday([{ date: '2026-10-12' }, { date: '2026-10-13' }], evening)).toBe(true);
    // 23:30 UTC on the 12th is already the 13th in Vietnam.
    expect(hasEntryToday([{ date: '2026-10-13' }], new Date('2026-10-12T23:30:00Z'))).toBe(true);
  });
});
