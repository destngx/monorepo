import { describe, expect, it } from 'vitest';
import { gaugeWindows, leftColor, leftPercent, resetsIn, usageColor, windowLabel } from './usage';

describe('windowLabel', () => {
  it('names the known windows and passes others through', () => {
    expect(windowLabel({ name: '5h' })).toBe('5-hour window');
    expect(windowLabel({ name: 'weekly' })).toBe('Weekly window');
    expect(windowLabel({ name: 'monthly' })).toBe('monthly');
  });
});

describe('usageColor', () => {
  it('warns from 70% and alerts from 90%', () => {
    expect(usageColor(69.9)).toBe('bg-emerald-500');
    expect(usageColor(70)).toBe('bg-amber-400');
    expect(usageColor(90)).toBe('bg-rose-500');
  });
});

describe('resetsIn', () => {
  const now = new Date('2026-10-13T10:00:00Z');

  it('counts down in days, hours and minutes', () => {
    expect(resetsIn('2026-10-13T10:45:00Z', now)).toBe('in 45m');
    expect(resetsIn('2026-10-13T12:15:00Z', now)).toBe('in 2h 15m');
    expect(resetsIn('2026-10-13T13:00:00Z', now)).toBe('in 3h');
    expect(resetsIn('2026-10-15T13:20:00Z', now)).toBe('in 2d 3h');
    expect(resetsIn('2026-10-15T10:00:00Z', now)).toBe('in 2d');
  });

  it('rounds partial minutes up so a near reset never shows 0m', () => {
    expect(resetsIn('2026-10-13T10:00:20Z', now)).toBe('in 1m');
  });

  it('reports a past reset as due and an unknown one as empty', () => {
    expect(resetsIn('2026-10-13T09:00:00Z', now)).toBe('due now');
    expect(resetsIn(undefined, now)).toBe('');
    expect(resetsIn('not a date', now)).toBe('');
  });
});

describe('leftPercent', () => {
  it('is what remains, clamped to 0-100', () => {
    expect(leftPercent({ usedPercent: 12 })).toBe(88);
    expect(leftPercent({ usedPercent: 130 })).toBe(0);
    expect(leftPercent({ usedPercent: -5 })).toBe(100);
    expect(leftPercent({ usedPercent: Number.NaN })).toBe(0);
  });
});

describe('leftColor', () => {
  it('runs from red with nothing left to green with everything left', () => {
    expect(leftColor(0)).toBe('oklch(0.7 0.17 25.0)');
    expect(leftColor(50)).toBe('oklch(0.7 0.17 87.5)');
    expect(leftColor(100)).toBe('oklch(0.7 0.17 150.0)');
    expect(leftColor(140)).toBe(leftColor(100));
  });
});

describe('gaugeWindows', () => {
  it('picks the weekly and 5-hour windows by name', () => {
    const weekly = { name: 'weekly', usedPercent: 40 };
    const fiveHour = { name: '5h', usedPercent: 12 };
    expect(gaugeWindows([fiveHour, weekly])).toEqual({ weekly, fiveHour });
    expect(gaugeWindows([])).toEqual({ weekly: null, fiveHour: null });
  });
});
