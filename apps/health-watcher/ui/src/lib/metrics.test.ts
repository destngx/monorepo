import { describe, expect, it } from 'vitest';
import type { Metric, TrendPoint } from '$lib/api/types';
import { formatChange, formatNumber, pointRange, pointSource, pointValue } from './metrics';

const glucose: Metric = {
  code: 'glucose',
  loinc: '',
  name: 'Glucose',
  nameVi: '',
  category: 'glucose',
  unit: 'mmol/L',
  aliases: [],
};
const point = (p: Partial<TrendPoint>): TrendPoint => ({ date: '2025-01-01', printed: '', source: 'paper', ...p });

describe('metric formatting', () => {
  it('keeps 4 significant digits below 1000 and whole numbers above', () => {
    expect(formatNumber(6.993783)).toBe('6.994');
    expect(formatNumber(0.85)).toBe('0.85');
    expect(formatNumber(1.025)).toBe('1.025');
    expect(formatNumber(145)).toBe('145');
    expect(formatNumber(12345.6)).toBe('12346');
  });

  it('signs a change with a real minus', () => {
    expect(formatChange(1.2)).toBe('+1.2');
    expect(formatChange(-0.904885)).toBe('−0.9049');
    expect(formatChange(0)).toBe('0');
  });

  it('shows the converted value, or the printed one when there is no number', () => {
    expect(pointValue(point({ value: 6.993783, printed: '126', printedUnit: 'mg/dL' }), glucose)).toEqual({
      value: '6.994',
      unit: 'mmol/L',
    });
    expect(pointValue(point({ printed: 'Âm tính' }), glucose)).toEqual({ value: 'Âm tính', unit: '' });
  });

  it('writes the paper range in the canonical unit, else as printed', () => {
    expect(pointRange(point({ refLow: 3.9, refHigh: 6.1 }))).toBe('3.9–6.1');
    expect(pointRange(point({ refHigh: 5.2 }))).toBe('< 5.2');
    expect(pointRange(point({ refLow: 1 }))).toBe('> 1');
    expect(pointRange(point({ refText: 'Âm tính' }))).toBe('Âm tính');
    expect(pointRange(point({}))).toBe('');
  });

  it('names where a point came from', () => {
    expect(pointSource(point({ provider: 'Lab A' }))).toBe('Lab A');
    expect(pointSource(point({ source: 'typed' }))).toBe('Typed in');
    expect(pointSource(point({ source: 'computed' }))).toBe('Computed');
  });
});
