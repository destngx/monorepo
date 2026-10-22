import type { Metric, TrendPoint } from '$lib/api/types';

const sig = new Intl.NumberFormat('en-US', { maximumSignificantDigits: 4, useGrouping: false });
const whole = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0, useGrouping: false });

/** A measured number with up to 4 significant digits, whole from 1000 up: 6.994, 0.85, 145, 12345. */
export function formatNumber(value: number): string {
  return Math.abs(value) >= 1000 ? whole.format(value) : sig.format(value);
}

/** A change with its sign: +1.2, −0.9, 0. */
export function formatChange(change: number): string {
  if (change === 0) return '0';
  return `${change > 0 ? '+' : '−'}${formatNumber(Math.abs(change))}`;
}

/** A point's value in the metric's unit, or as printed when it has no number on the chart. */
export function pointValue(point: TrendPoint, metric: Metric): { value: string; unit: string } {
  if (point.value !== undefined) return { value: formatNumber(point.value), unit: metric.unit };
  return { value: point.printed, unit: point.printedUnit ?? '' };
}

/** The paper's range in the metric's unit (`3.9–6.1`, `< 5.2`, `> 1`), else as printed; '' without one. */
export function pointRange(point: TrendPoint): string {
  const { refLow: low, refHigh: high } = point;
  if (low !== undefined && high !== undefined) return `${formatNumber(low)}–${formatNumber(high)}`;
  if (high !== undefined) return `< ${formatNumber(high)}`;
  if (low !== undefined) return `> ${formatNumber(low)}`;
  return point.refText ?? '';
}

/** Where a point came from, in words. */
export function pointSource(point: TrendPoint): string {
  if (point.source === 'computed') return 'Computed';
  if (point.source === 'typed') return 'Typed in';
  return point.provider || 'Paper';
}

/** The vitals that can be typed in (HW-MET-2), in form order; BMI is computed from weight and height. */
export const typedVitals = [
  'weight',
  'height',
  'bp_systolic',
  'bp_diastolic',
  'heart_rate',
  'temperature',
  'spo2',
  'waist',
];
