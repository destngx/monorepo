import { describe, expect, it } from 'vitest';
import { DIRECTIONS, natureLabel, spendShape } from './trend';

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

describe('natureLabel', () => {
  it('names needs and wants', () => {
    expect(natureLabel('fixed')).toBe('Fixed cost');
    expect(natureLabel('essential')).toBe('Need');
    expect(natureLabel('flexible')).toBe('Want');
  });

  it('calls a missing or lending nature untagged', () => {
    expect(natureLabel(undefined)).toBe('Untagged');
    expect(natureLabel('lending')).toBe('Untagged');
  });
});

describe('DIRECTIONS', () => {
  it('labels every move in words', () => {
    expect(Object.values(DIRECTIONS).map((direction) => direction.label)).toEqual(['Increase', 'Lower', 'Keep']);
  });
});
