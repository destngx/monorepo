import { describe, expect, it } from 'vitest';
import type { Category } from '$lib/api/types';
import { groupCategories, NATURE_OPTIONS, natureChoice, parseSplitTags, withSplitTag } from './natures';

describe('category natures', () => {
  it('offers every nature the engine accepts, lending included', () => {
    expect(NATURE_OPTIONS.map((option) => option.value)).toEqual(['none', 'fixed', 'essential', 'flexible', 'lending']);
  });

  it('groups categories by type, expense first and names sorted', () => {
    const categories: Category[] = [
      { name: 'Salary', type: 'income' },
      { name: 'Rent', type: 'expense', nature: 'fixed' },
      { name: 'Lending / Repayment', type: 'non-budget', nature: 'lending' },
      { name: 'Food', type: 'expense' },
    ];
    expect(groupCategories(categories).map((group) => [group.label, group.categories.map((c) => c.name)])).toEqual([
      ['Expense', ['Food', 'Rent']],
      ['Income', ['Salary']],
      ['Non-budget', ['Lending / Repayment']],
    ]);
  });

  it('shows an untagged category as none', () => {
    expect(natureChoice({})).toBe('none');
    expect(natureChoice({ nature: 'lending' })).toBe('lending');
  });

  it('reads split tags typed as a comma list the way the engine stores them', () => {
    expect(parseSplitTags(' Eating-Out, ,takeaway,eating-out ')).toEqual(['eating-out', 'takeaway']);
    expect(parseSplitTags('')).toEqual([]);
  });

  it('adds a suggested split tag once and keeps the card tag', () => {
    expect(withSplitTag(['silver'], 'eating-out')).toEqual(['silver', 'eating-out']);
    expect(withSplitTag(['silver', 'Eating-Out'], 'eating-out')).toEqual(['silver', 'Eating-Out']);
    const tags = ['silver'];
    expect(withSplitTag(tags, undefined)).toBe(tags);
  });
});
