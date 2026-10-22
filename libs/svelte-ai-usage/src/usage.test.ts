import { describe, expect, it } from 'vitest';
import { gaugeWindows, leftColorScale, leftPercent, resetsIn, windowLabel } from './usage';

describe('windowLabel', () => {
  it('Given the gateway windows, when labelled, then they read as 5-hour and weekly and others pass through', () => {
    expect(windowLabel({ name: '5h' })).toBe('5-hour window');
    expect(windowLabel({ name: 'weekly' })).toBe('Weekly window');
    expect(windowLabel({ name: 'monthly' })).toBe('monthly');
  });
});

describe('resetsIn', () => {
  const now = new Date('2026-10-13T10:00:00Z');

  it('Given a reset time, when counted from now, then it reads in days, hours or minutes', () => {
    expect(resetsIn('2026-10-13T10:45:00Z', now)).toBe('in 45m');
    expect(resetsIn('2026-10-13T12:15:00Z', now)).toBe('in 2h 15m');
    expect(resetsIn('2026-10-13T13:00:00Z', now)).toBe('in 3h');
    expect(resetsIn('2026-10-15T13:20:00Z', now)).toBe('in 2d 3h');
    expect(resetsIn('2026-10-15T10:00:00Z', now)).toBe('in 2d');
  });

  it('Given a reset seconds away, when counted, then partial minutes round up so it never shows 0m', () => {
    expect(resetsIn('2026-10-13T10:00:20Z', now)).toBe('in 1m');
  });

  it('Given a past or unknown reset, when counted, then it is due now or empty', () => {
    expect(resetsIn('2026-10-13T09:00:00Z', now)).toBe('due now');
    expect(resetsIn(undefined, now)).toBe('');
    expect(resetsIn('not a date', now)).toBe('');
  });
});

describe('leftPercent', () => {
  it('Given a used share, when turned into what is left, then it stays within 0 to 100', () => {
    expect(leftPercent({ usedPercent: 12 })).toBe(88);
    expect(leftPercent({ usedPercent: 130 })).toBe(0);
    expect(leftPercent({ usedPercent: -5 })).toBe(100);
    expect(leftPercent({ usedPercent: Number.NaN })).toBe(0);
  });
});

describe('leftColorScale', () => {
  it('Given a scale, when coloured, then it runs from the empty hue to the full hue at even lightness', () => {
    const leftColor = leftColorScale({ lightness: 0.7, chroma: 0.17, emptyHue: 25, fullHue: 150 });
    expect(leftColor(0)).toBe('oklch(0.7 0.17 25.0)');
    expect(leftColor(50)).toBe('oklch(0.7 0.17 87.5)');
    expect(leftColor(100)).toBe('oklch(0.7 0.17 150.0)');
    expect(leftColor(140)).toBe(leftColor(100));
    expect(leftColor(-10)).toBe(leftColor(0));
  });
});

describe('gaugeWindows', () => {
  it('Given the windows, when picked for the gauge, then weekly is outer and 5-hour inner', () => {
    const weekly = { name: 'weekly', usedPercent: 40 };
    const fiveHour = { name: '5h', usedPercent: 12 };
    expect(gaugeWindows([fiveHour, weekly])).toEqual({ weekly, fiveHour });
    expect(gaugeWindows([])).toEqual({ weekly: null, fiveHour: null });
  });
});
