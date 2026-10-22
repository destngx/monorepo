import { describe, expect, it } from 'vitest';
import { formatReset, leftColor } from './aiUsage';

describe('AI quota presentation', () => {
  it('Given what is left, when coloured, then full is green and empty is red', () => {
    expect(leftColor(100)).toBe('oklch(0.58 0.09 145.0)');
    expect(leftColor(0)).toBe('oklch(0.58 0.09 30.0)');
  });

  it('Given a reset time, when shown, then it is the Vietnam day and time', () => {
    expect(formatReset('2026-10-22T15:04:00Z')).toBe('22/10 22:04');
    expect(formatReset('soon')).toBe('');
  });
});
