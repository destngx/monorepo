import { describe, expect, it } from 'vitest';
import { formatReset, gaugeWindows, leftColor, leftPercent, resetsIn, windowLabel } from './aiUsage';

describe('AI quota presentation', () => {
  it('Given the gateway windows, when labelled, then they read as 5-hour and weekly', () => {
    expect(windowLabel({ name: '5h' })).toBe('5-hour window');
    expect(windowLabel({ name: 'weekly' })).toBe('Weekly window');
    expect(windowLabel({ name: 'daily' })).toBe('daily');
  });

  it('Given a used share, when turned into what is left, then it stays within 0 to 100', () => {
    expect(leftPercent({ usedPercent: 12 })).toBe(88);
    expect(leftPercent({ usedPercent: 130 })).toBe(0);
    expect(leftPercent({ usedPercent: -5 })).toBe(100);
    expect(leftPercent({ usedPercent: Number.NaN })).toBe(0);
  });

  it('Given what is left, when coloured, then full is green and empty is red', () => {
    expect(leftColor(100)).toBe('oklch(0.7 0.17 150.0)');
    expect(leftColor(0)).toBe('oklch(0.7 0.17 25.0)');
  });

  it('Given a reset time, when counted from now, then it reads in days, hours or minutes', () => {
    const now = new Date('2026-10-22T10:00:00Z');
    expect(resetsIn('2026-10-24T13:00:00Z', now)).toBe('in 2d 3h');
    expect(resetsIn('2026-10-22T11:30:00Z', now)).toBe('in 1h 30m');
    expect(resetsIn('2026-10-22T10:45:00Z', now)).toBe('in 45m');
    expect(resetsIn('2026-10-22T09:00:00Z', now)).toBe('due now');
    expect(resetsIn(undefined, now)).toBe('');
    expect(resetsIn('soon', now)).toBe('');
  });

  it('Given a reset time, when shown, then it is the Vietnam day and time', () => {
    expect(formatReset('2026-10-22T15:04:00Z')).toBe('22/10 22:04');
    expect(formatReset('soon')).toBe('');
  });

  it('Given the windows, when picked for the gauge, then weekly is outer and 5-hour inner', () => {
    const weekly = { name: 'weekly', usedPercent: 40 };
    const fiveHour = { name: '5h', usedPercent: 12 };
    expect(gaugeWindows([fiveHour, weekly])).toEqual({ weekly, fiveHour });
    expect(gaugeWindows([])).toEqual({ weekly: null, fiveHour: null });
  });
});
