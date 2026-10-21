import { describe, expect, it } from 'vitest';
import { ageOn, flagLabel, formatDate, formatSize, isImage, options, todayISO, visitTypeLabels } from './format';

describe('formatDate', () => {
  it('shows ISO dates day first', () => {
    expect(formatDate('2025-03-04')).toBe('04/03/2025');
    expect(formatDate('2025-03-04T10:00:00Z')).toBe('04/03/2025');
  });
  it('is empty for missing or malformed dates', () => {
    expect(formatDate(undefined)).toBe('');
    expect(formatDate('04/03/2025')).toBe('');
  });
});

describe('todayISO', () => {
  it('uses the Vietnam calendar day, not UTC', () => {
    expect(todayISO(new Date('2026-10-20T18:30:00Z'))).toBe('2026-10-21');
  });
});

describe('ageOn', () => {
  it('counts whole years, before and after the birthday', () => {
    expect(ageOn('2019-05-10', '2026-05-09')).toBe(6);
    expect(ageOn('2019-05-10', '2026-05-10')).toBe(7);
  });
  it('is null for unknown or future birth dates', () => {
    expect(ageOn(undefined, '2026-01-01')).toBeNull();
    expect(ageOn('2030-01-01', '2026-01-01')).toBeNull();
  });
});

describe('formatSize', () => {
  it('picks a unit', () => {
    expect(formatSize(500)).toBe('500 B');
    expect(formatSize(2048)).toBe('2 KB');
    expect(formatSize(3.5 * 1024 * 1024)).toBe('3.5 MB');
  });
});

describe('options and isImage', () => {
  it('keeps the declared order', () => {
    expect(options(visitTypeLabels)[0]).toEqual({ value: 'checkup', label: 'Check-up' });
  });
  it('treats HEIC as not viewable in the browser', () => {
    expect(isImage('image/jpeg')).toBe(true);
    expect(isImage('image/heic')).toBe(false);
    expect(isImage('application/pdf')).toBe(false);
  });
});

describe('flagLabel', () => {
  it('names the printed flags', () => {
    expect(flagLabel('H')).toBe('High');
    expect(flagLabel('L')).toBe('Low');
    expect(flagLabel('*')).toBe('Abnormal');
    expect(flagLabel('')).toBe('');
  });
});
