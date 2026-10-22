import { describe, expect, it } from 'vitest';
import { formatDate, formatDateTime, formatSize, MISSING, todayISO } from './format';

describe('formatDate', () => {
  it('Given a calendar date When formatted Then it reads dd/MM/yyyy', () => {
    expect(formatDate('2026-10-22')).toBe('22/10/2026');
  });

  it('Given a date with a time and zone When formatted Then only the calendar part counts', () => {
    expect(formatDate('2026-10-22T23:30:00-12:00')).toBe('22/10/2026');
  });

  it('Given no date When formatted Then it shows the missing mark; other text stays as it is', () => {
    expect(formatDate(null)).toBe(MISSING);
    expect(formatDate('')).toBe(MISSING);
    expect(formatDate('Pending')).toBe('Pending');
  });
});

describe('formatDateTime', () => {
  it('Given an instant When formatted in a zone Then it reads dd/MM/yyyy HH:mm on a 24-hour clock', () => {
    expect(formatDateTime('2026-10-22T15:05:00Z', 'Asia/Ho_Chi_Minh')).toBe('22/10/2026 22:05');
    expect(formatDateTime('2026-10-22T00:05:00Z', 'UTC')).toBe('22/10/2026 00:05');
  });

  it('Given an unreadable instant When formatted Then it shows the missing mark', () => {
    expect(formatDateTime('soon')).toBe(MISSING);
    expect(formatDateTime(undefined)).toBe(MISSING);
  });
});

describe('todayISO', () => {
  it('Given an instant late in the UTC day When asked Then it is already tomorrow in Vietnam', () => {
    expect(todayISO(new Date('2026-10-22T18:00:00Z'))).toBe('2026-10-23');
  });
});

describe('formatSize', () => {
  it('Given byte counts When formatted Then they read in B, KB or MB', () => {
    expect(formatSize(512)).toBe('512 B');
    expect(formatSize(2048)).toBe('2 KB');
    expect(formatSize(5 * 1024 * 1024)).toBe('5.0 MB');
    expect(formatSize(Number.NaN)).toBe(MISSING);
  });
});
