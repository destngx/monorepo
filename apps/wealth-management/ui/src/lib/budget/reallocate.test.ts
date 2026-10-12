import { describe, expect, it } from 'vitest';
import type { BudgetLine } from '$lib/api/types';
import { reallocationPreview, reallocationProblems } from './reallocate';

const line = (category: string, limit: number) => ({ category, limit }) as BudgetLine;
const lines = [line('Food', 3_000_000), line('Fun', 1_000_000)];

describe('reallocation', () => {
  it('accepts a move within the source limit', () => {
    const draft = { from: 'Food', to: 'Fun', amount: 500_000 };
    expect(reallocationProblems(lines, draft)).toEqual([]);
    expect(reallocationPreview(lines, draft)).toEqual({ from: 2_500_000, to: 1_500_000 });
  });

  it('allows moving the whole limit', () => {
    expect(reallocationProblems(lines, { from: 'Fun', to: 'Food', amount: 1_000_000 })).toEqual([]);
  });

  it('rejects missing or identical categories', () => {
    expect(reallocationProblems(lines, { from: '', to: 'Fun', amount: 1 })).toContain(
      'Choose the category to take from.',
    );
    expect(reallocationProblems(lines, { from: 'Food', to: 'Rent', amount: 1 })).toContain(
      'Choose the category to give to.',
    );
    expect(reallocationProblems(lines, { from: 'Food', to: 'Food', amount: 1 })).toContain(
      'Choose two different categories.',
    );
  });

  it('rejects zero, negative and oversized amounts without quoting figures', () => {
    expect(reallocationProblems(lines, { from: 'Food', to: 'Fun', amount: 0 })).toContain(
      'Enter an amount above zero.',
    );
    expect(reallocationProblems(lines, { from: 'Food', to: 'Fun', amount: -5 })).toContain(
      'Enter an amount above zero.',
    );
    expect(reallocationProblems(lines, { from: 'Food', to: 'Fun', amount: null })).toContain(
      'Enter an amount above zero.',
    );
    const tooMuch = reallocationProblems(lines, { from: 'Fun', to: 'Food', amount: 1_000_001 });
    expect(tooMuch).toEqual(['The amount is more than the limit of Fun.']);
    expect(reallocationPreview(lines, { from: 'Fun', to: 'Food', amount: 1_000_001 })).toBeNull();
  });
});
