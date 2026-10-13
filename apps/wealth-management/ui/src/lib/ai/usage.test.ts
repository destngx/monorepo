import { describe, expect, it } from 'vitest';
import { resetsIn, usageColor, windowLabel } from './usage';

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
    expect(resetsIn('2026-10-13T10:45:00Z', now)).toBe('resets in 45m');
    expect(resetsIn('2026-10-13T12:15:00Z', now)).toBe('resets in 2h 15m');
    expect(resetsIn('2026-10-13T13:00:00Z', now)).toBe('resets in 3h');
    expect(resetsIn('2026-10-15T13:20:00Z', now)).toBe('resets in 2d 3h');
    expect(resetsIn('2026-10-15T10:00:00Z', now)).toBe('resets in 2d');
  });

  it('rounds partial minutes up so a near reset never shows 0m', () => {
    expect(resetsIn('2026-10-13T10:00:20Z', now)).toBe('resets in 1m');
  });

  it('reports a past reset as due and an unknown one as empty', () => {
    expect(resetsIn('2026-10-13T09:00:00Z', now)).toBe('reset due');
    expect(resetsIn(undefined, now)).toBe('');
    expect(resetsIn('not a date', now)).toBe('');
  });
});
