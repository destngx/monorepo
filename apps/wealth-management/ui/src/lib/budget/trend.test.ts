import { describe, expect, it } from 'vitest';
import { spendShape } from './trend';

describe('spendShape', () => {
  it('names the trend and an uneven pattern', () => {
    expect(spendShape('rising', 'variable')).toBe('Rising, swings month to month');
    expect(spendShape('falling', 'occasional')).toBe('Falling, only some months');
  });

  it('leaves out a steady pattern and an unknown trend', () => {
    expect(spendShape('steady', 'steady')).toBe('Steady');
    expect(spendShape('unknown', 'occasional')).toBe('Only some months');
    expect(spendShape('unknown', 'steady')).toBe('');
  });
});
