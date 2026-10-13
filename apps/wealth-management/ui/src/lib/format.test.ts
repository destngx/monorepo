import { describe, expect, it } from 'vitest';
import {
  formatDate,
  formatDateTime,
  formatMonth,
  formatPercent,
  formatSignedPercent,
  formatVND,
  shiftMonth,
  todayISO,
} from './format';

describe('formatVND', () => {
  it('groups with dots and rounds to whole dong', () => {
    expect(formatVND(1250000.6)).toBe('1.250.001 ₫');
  });

  it('never renders negative zero', () => {
    expect(formatVND(-0.2)).toBe('0 ₫');
  });

  it('renders missing values as a dash', () => {
    expect(formatVND(null)).toBe('-');
    expect(formatVND(Number.NaN)).toBe('-');
  });
});

describe('formatDateTime', () => {
  it('shows an instant as dd/MM/yyyy HH:mm in the given timezone', () => {
    expect(formatDateTime('2026-10-13T15:04:00Z', 'UTC')).toBe('13/10/2026 15:04');
    expect(formatDateTime('2026-10-13T20:30:00Z', 'Asia/Ho_Chi_Minh')).toBe('14/10/2026 03:30');
  });

  it('shows a dash for missing or unreadable values', () => {
    expect(formatDateTime(undefined)).toBe('-');
    expect(formatDateTime('not a date')).toBe('-');
  });
});

describe('formatDate', () => {
  it('uses the calendar date of a timestamp regardless of timezone', () => {
    expect(formatDate('2026-03-01T00:00:00Z')).toBe('01/03/2026');
  });

  it('passes through values it cannot parse', () => {
    expect(formatDate('5, 30')).toBe('5, 30');
  });
});

describe('months', () => {
  it('shifts across year boundaries', () => {
    expect(shiftMonth('2026-01', -1)).toBe('2025-12');
    expect(shiftMonth('2026-12', 1)).toBe('2027-01');
  });

  it('labels a month', () => {
    expect(formatMonth('2026-10')).toBe('Oct 2026');
    expect(formatMonth('2026-10', true)).toBe('Oct');
  });

  it('uses Vietnam time for today', () => {
    // 20:00 UTC on 11 Oct is already 12 Oct in Ho Chi Minh City.
    expect(todayISO(new Date('2026-10-11T20:00:00Z'))).toBe('2026-10-12');
  });
});

describe('percentages', () => {
  it('signs positive changes', () => {
    expect(formatSignedPercent(12.345)).toBe('+12.3%');
    expect(formatSignedPercent(-4)).toBe('-4.0%');
    expect(formatPercent(null)).toBe('-');
  });
});
