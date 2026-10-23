import { fireEvent, render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { describe, expect, it } from 'vitest';
import Card from './Card.svelte';

const text = (value: string) => createRawSnippet(() => ({ render: () => `<p>${value}</p>` }));

describe('Card', () => {
  it('Given a plain card When shown Then the title, actions and body show with no fold', () => {
    render(Card, { title: 'Plain', actions: text('Action'), children: text('Body') });

    expect(screen.getByRole('heading', { name: 'Plain' })).toBeTruthy();
    expect(screen.getByText('Action')).toBeTruthy();
    expect(screen.getByText('Body')).toBeTruthy();
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('Given a foldable card that starts folded When opened Then the body and actions show', async () => {
    render(Card, { title: 'Tools', foldable: true, open: false, actions: text('Action'), children: text('Body') });

    const toggle = screen.getByRole('button', { name: 'Open Tools' });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText('Action')).toBeNull();
    expect(screen.getByText('Body').closest('[hidden]')).toBeTruthy();

    await fireEvent.click(toggle);

    expect(screen.getByRole('button', { name: 'Fold Tools' }).getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText('Action')).toBeTruthy();
    expect(screen.getByText('Body').closest('[hidden]')).toBeNull();
  });
});
