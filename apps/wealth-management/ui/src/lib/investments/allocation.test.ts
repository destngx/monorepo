import type { Allocation, AllocationLine } from '$lib/api/types';
import { describe, expect, it } from 'vitest';
import {
  allocationSlices,
  allocationStatus,
  breachSummary,
  lineStatus,
  parseBrowserTargets,
  targetsError,
  targetValues,
} from './allocation';

const line = (overrides: Partial<AllocationLine>): AllocationLine => ({
  class: 'cash',
  label: 'Cash and deposits',
  value: 0,
  share: 0,
  targeted: true,
  current: 0,
  target: null,
  drift: null,
  status: 'no_target',
  ...overrides,
});

const allocation = (overrides: Partial<Allocation> = {}): Allocation => ({
  lines: [
    line({ class: 'cash', value: 60, current: 60, target: 50, drift: 10, status: 'over' }),
    line({
      class: 'funds',
      label: 'Fund certificates',
      value: 30,
      current: 30,
      target: 40,
      drift: -10,
      status: 'under',
    }),
    line({ class: 'crypto', label: 'Crypto', value: 10, current: 10, target: 10, drift: 0, status: 'on_target' }),
    line({ class: 'moneyLent', label: 'Money lent', value: 0, targeted: false, current: null }),
  ],
  total: 100,
  targetedTotal: 100,
  targets: { cash: 50, funds: 40, crypto: 10 },
  driftThreshold: 5,
  rebalance: true,
  unvalued: 0,
  issues: [],
  generatedAt: '2026-10-17T00:00:00Z',
  ...overrides,
});

describe('allocation status', () => {
  it('names each line status with a tone and a label', () => {
    expect(lineStatus('over')).toEqual({ tone: 'warning', label: 'Over target' });
    expect(lineStatus('under')).toEqual({ tone: 'warning', label: 'Under target' });
    expect(lineStatus('on_target')).toEqual({ tone: 'good', label: 'On target' });
    expect(lineStatus('no_target')).toEqual({ tone: 'neutral', label: 'No target' });
  });

  it('reads the headline from the engine rebalance flag', () => {
    expect(allocationStatus(allocation()).label).toBe('Rebalance suggested');
    expect(allocationStatus(allocation({ rebalance: false })).tone).toBe('good');
    expect(allocationStatus(allocation({ targets: null, rebalance: false })).label).toBe('No targets set');
  });

  it('summarises the classes beyond the threshold', () => {
    expect(breachSummary(allocation())).toEqual([
      'Cash and deposits is over target by 10.0 points',
      'Fund certificates is under target by 10.0 points',
    ]);
  });

  it('draws only lines with a value, each class in its own colour', () => {
    const slices = allocationSlices(allocation());
    expect(slices.map((slice) => slice.label)).toEqual(['Cash and deposits', 'Fund certificates', 'Crypto']);
    expect(new Set(slices.map((slice) => slice.color)).size).toBe(3);
  });
});

describe('targets', () => {
  it('requires the targets to add up to 100%', () => {
    expect(targetsError({ cash: 30, funds: 50, crypto: 20 })).toBeNull();
    expect(targetsError({ cash: 30, funds: 50, crypto: 10 })).toBe('Targets must add up to 100% (now 90%).');
    expect(targetsError({ cash: 130, funds: -50, crypto: 20 })).toMatch(/between 0 and 100/);
  });

  it('saves the three targets together, or clears them', () => {
    expect(targetValues({ cash: 60, funds: 30, crypto: 10 })).toEqual([
      { key: 'allocation.target.cash', value: 60 },
      { key: 'allocation.target.funds', value: 30 },
      { key: 'allocation.target.crypto', value: 10 },
    ]);
    expect(targetValues(null).every((value) => value.value === null)).toBe(true);
  });

  it('reads targets an older version kept in the browser, ignoring invalid ones', () => {
    expect(parseBrowserTargets('{"cash":30,"funds":50,"crypto":20}')).toEqual({ cash: 30, funds: 50, crypto: 20 });
    expect(parseBrowserTargets('')).toBeNull();
    expect(parseBrowserTargets('not json')).toBeNull();
    expect(parseBrowserTargets('{"cash":30,"funds":50}')).toBeNull();
    expect(parseBrowserTargets('{"cash":30,"funds":50,"crypto":10}')).toBeNull();
  });
});
