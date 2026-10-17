import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MASK_TOKEN_LABEL } from '$lib/components/maskTokens';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import Markdown from './Markdown.svelte';

afterEach(() => mask.hide());

describe('Markdown', () => {
  it('Given model prose with amounts When masked Then each amount is a token button inside the markup', () => {
    const { container } = render(Markdown, { text: 'You spent **1.500.000 ₫** on food, 20% more' });

    const token = screen.getByRole('button', { name: MASK_TOKEN_LABEL });
    expect(token.closest('strong')).toBeTruthy();
    expect(token.textContent).toBe(MASK);
    expect(container.textContent).toBe(`You spent ${MASK} on food, 20% more`);
  });

  it('Given a masked amount When its token is clicked Then amounts reveal globally and the prose is the original', async () => {
    const { container } = render(Markdown, { text: 'You spent **1.500.000 ₫** on food' });

    await fireEvent.click(screen.getByRole('button', { name: MASK_TOKEN_LABEL }));

    expect(mask.revealed).toBe(true);
    expect(container.textContent).toBe('You spent 1.500.000 ₫ on food');
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('Given an amount in a link label in a clickable card When its token is clicked Then the mask toggles without navigating', async () => {
    const card = document.createElement('div');
    const cardClicked = vi.fn();
    card.addEventListener('click', cardClicked);
    document.body.append(card);
    render(Markdown, { target: card, props: { text: 'See [the 300k plan](https://example.com/plan)' } });
    const link = card.querySelector('a') as HTMLAnchorElement;

    // No button nested in the link: the token there is a pointer-only span.
    expect(link.querySelector('button')).toBeNull();
    const isDefaultAllowed = await fireEvent.click(link.querySelector('[data-mask-toggle]') as HTMLElement);

    expect(isDefaultAllowed).toBe(false);
    expect(cardClicked).not.toHaveBeenCalled();
    expect(mask.revealed).toBe(true);
    expect(card.querySelector('a')?.textContent).toBe('the 300k plan');
    card.remove();
  });

  it('Given hostile markup next to an amount When masked Then it stays escaped and only the token is markup', () => {
    const { container } = render(Markdown, { text: '<img src=x onerror=alert(1)> 1.000.000 ₫' });
    expect(container.querySelector('img')).toBeNull();
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(container.textContent).toBe(`<img src=x onerror=alert(1)> ${MASK}`);
  });
});
