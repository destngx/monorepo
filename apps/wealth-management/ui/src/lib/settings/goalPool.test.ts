import { describe, expect, it } from 'vitest';
import { goalPoolChanges, goalPoolError } from './goalPool';

const saved = { emergencyMonths: 6, spendingBufferMonths: 1 };

describe('goal pool form', () => {
  it('sends only the keys that changed', () => {
    expect(goalPoolChanges(saved, { emergencyMonths: '6', spendingBufferMonths: '1' })).toEqual([]);
    expect(goalPoolChanges(saved, { emergencyMonths: 9, spendingBufferMonths: '1' })).toEqual([
      { key: 'goal.emergencyMonths', value: 9 },
    ]);
  });

  it('clears a key back to its default when the field is emptied', () => {
    expect(
      goalPoolChanges(
        { emergencyMonths: 9, spendingBufferMonths: 1 },
        { emergencyMonths: '', spendingBufferMonths: '1' },
      ),
    ).toEqual([{ key: 'goal.emergencyMonths', value: null }]);
  });

  it('checks the ranges the engine accepts', () => {
    expect(goalPoolError({ emergencyMonths: '6', spendingBufferMonths: '0' })).toBeNull();
    expect(goalPoolError({ emergencyMonths: '0', spendingBufferMonths: '1' })).toBe(
      'Emergency fund must be between 1 and 24 months.',
    );
    expect(goalPoolError({ emergencyMonths: '6', spendingBufferMonths: '13' })).toMatch(/Spending buffer/);
    expect(goalPoolError({ emergencyMonths: 'x', spendingBufferMonths: '1' })).toMatch(/Emergency fund/);
  });
});
