import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import type { Category } from '$lib/api/types';

vi.mock('$lib/api/endpoints', () => ({ api: {} }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

import CategoryNatureCard from './CategoryNatureCard.svelte';

const categories = [
  { name: 'Groceries', type: 'expense', tags: ['essential'] },
  { name: 'Dining', type: 'expense', tags: ['flexible'] },
] as unknown as Category[];

describe('CategoryNatureCard', () => {
  it('Given the budget page opens When the card renders Then it is folded with an Open button', async () => {
    render(CategoryNatureCard, { categories });

    const toggle = screen.getByRole('button', { name: 'Open Categories' });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByRole('combobox', { name: 'Groceries' })).toBeNull();

    await fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Fold Categories' }).getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByRole('combobox', { name: 'Groceries' })).toBeTruthy();
  });
});
