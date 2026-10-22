import { describe, expect, it } from 'vitest';
import { leftColor, usageColor } from './usage';

describe('usageColor', () => {
  it('warns from 70% and alerts from 90%', () => {
    expect(usageColor(69.9)).toBe('bg-emerald-500');
    expect(usageColor(70)).toBe('bg-amber-400');
    expect(usageColor(90)).toBe('bg-rose-500');
  });
});

describe('leftColor', () => {
  it('runs from red with nothing left to green with everything left', () => {
    expect(leftColor(0)).toBe('oklch(0.7 0.17 25.0)');
    expect(leftColor(50)).toBe('oklch(0.7 0.17 87.5)');
    expect(leftColor(100)).toBe('oklch(0.7 0.17 150.0)');
  });
});
