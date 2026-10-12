import { describe, expect, it } from 'vitest';
import { classValues, computeDrift, parseTargets, serializeTargets, targetsError } from './allocation';

describe('targets', () => {
  it('round-trips through storage and rejects invalid values', () => {
    const targets = { cash: 30, funds: 50, crypto: 20 };
    expect(parseTargets(serializeTargets(targets))).toEqual(targets);
    expect(parseTargets('')).toBeNull();
    expect(parseTargets('not json')).toBeNull();
    expect(parseTargets('{"cash":30,"funds":50}')).toBeNull();
    expect(parseTargets('{"cash":-1,"funds":81,"crypto":20}')).toBeNull();
  });

  it('requires the targets to add up to 100%', () => {
    expect(targetsError({ cash: 30, funds: 50, crypto: 20 })).toBeNull();
    expect(targetsError({ cash: 30, funds: 50, crypto: 10 })).toBe('Targets must add up to 100% (now 90%).');
    expect(targetsError({ cash: 130, funds: -50, crypto: 20 })).toMatch(/between 0 and 100/);
  });
});

describe('computeDrift (WM-INV-20)', () => {
  const values = classValues({ vndAccounts: 400, funds: 400, crypto: 200 });

  it('reports current share and drift per class', () => {
    const { rows, total, shouldRebalance } = computeDrift(values, { cash: 40, funds: 40, crypto: 20 });
    expect(total).toBe(1000);
    expect(rows.map((r) => [r.key, r.currentPercent, r.drift])).toEqual([
      ['cash', 40, 0],
      ['funds', 40, 0],
      ['crypto', 20, 0],
    ]);
    expect(shouldRebalance).toBe(false);
  });

  it('alerts only when a class drifts more than 5 percentage points', () => {
    expect(computeDrift(values, { cash: 35, funds: 45, crypto: 20 }).shouldRebalance).toBe(false);
    const drifted = computeDrift(values, { cash: 30, funds: 50, crypto: 20 });
    expect(drifted.shouldRebalance).toBe(true);
    expect(drifted.rows.filter((r) => r.breach).map((r) => [r.key, r.drift])).toEqual([
      ['cash', 10],
      ['funds', -10],
    ]);
  });

  it('does not shouldRebalance on an empty portfolio', () => {
    const empty = computeDrift(classValues({ vndAccounts: 0, funds: 0, crypto: 0 }), {
      cash: 100,
      funds: 0,
      crypto: 0,
    });
    expect(empty.shouldRebalance).toBe(false);
    expect(empty.rows[0].currentPercent).toBe(0);
  });
});
