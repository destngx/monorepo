import type { Category, CategoryNature, CategoryType } from '$lib/api/types';

export type NatureChoice = CategoryNature | 'none';

/** The natures a category can take, with what each means (WM-LEND-5). */
export const NATURE_OPTIONS: { value: NatureChoice; label: string; detail: string }[] = [
  { value: 'none', label: 'None', detail: 'Not tagged' },
  { value: 'fixed', label: 'Fixed', detail: 'Set by a contract, e.g. rent' },
  { value: 'essential', label: 'Essential', detail: 'Needed, but can be lowered' },
  { value: 'flexible', label: 'Flexible', detail: 'Wants' },
  { value: 'lending', label: 'Lending', detail: 'Money lent and repaid; not income or expense' },
];

const TYPE_ORDER: { type: CategoryType; label: string }[] = [
  { type: 'expense', label: 'Expense' },
  { type: 'income', label: 'Income' },
  { type: 'non-budget', label: 'Non-budget' },
];

/** Categories by type (expense first), names sorted, empty groups dropped. */
export function groupCategories(categories: Category[]): { label: string; categories: Category[] }[] {
  return TYPE_ORDER.map(({ type, label }) => ({
    label,
    categories: categories.filter((category) => category.type === type).sort((a, b) => a.name.localeCompare(b.name)),
  })).filter((group) => group.categories.length > 0);
}

/** The select value for a category: its nature, or 'none'. */
export function natureChoice(category: Pick<Category, 'nature'>): NatureChoice {
  return category.nature ?? 'none';
}
