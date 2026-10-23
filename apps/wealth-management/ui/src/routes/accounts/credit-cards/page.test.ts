import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import { testBank, testCard } from '$lib/cards/cards.testdata';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
const calls = vi.hoisted(() => [] as string[]);
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get: (_, name) => () => {
        calls.push(String(name));
        return new Promise(() => undefined);
      },
    },
  ),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import CreditCardsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Credit cards page', () => {
  afterEach(() => {
    mask.hide();
    calls.length = 0;
  });

  it('Given the cards are still loading When the page renders Then the header shows with the cards skeleton', () => {
    render(CreditCardsPage, { data: { engine: never(), banks: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Credit Cards' })).toBeTruthy();
    expect(screen.getAllByText('Loading credit cards…').length).toBeGreaterThan(0);
  });

  it('Given the cards fail When they settle Then the error and Retry show and the recommendations still load', async () => {
    render(CreditCardsPage, {
      data: { engine: never(), banks: defer(Promise.reject(new Error('Cards tab is invalid'))) } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('credit cards');
    expect(alert.textContent).toContain('Cards tab is invalid');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(calls).toContain('recommendations');
  });

  it('Given cards When the page renders Then the best-card and programme tools sit folded below the cards', async () => {
    render(CreditCardsPage, { data: { engine: never(), banks: settled([testBank()]) } as never });
    await flush();

    const best = screen.getByRole('button', { name: 'Open Best card for a payment' });
    const programmes = screen.getByRole('button', { name: 'Open Compare programmes' });
    expect(best.getAttribute('aria-expanded')).toBe('false');
    expect(programmes.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByRole('button', { name: 'Compare' })).toBeNull();
    const history = screen.getByRole('heading', { name: 'Compare cards' });
    expect(history.compareDocumentPosition(best) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('Given cards When Compare is clicked Then the comparison button stays busy until it answers', async () => {
    render(CreditCardsPage, { data: { engine: never(), banks: settled([testBank()]) } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Open Compare programmes' }));
    const compare = screen.getByRole('button', { name: 'Compare' });
    await fireEvent.click(compare);
    await flush();

    expect(calls).toContain('comparePrograms');
    expect(compare.getAttribute('aria-busy')).toBe('true');
    expect(compare.textContent).toContain('Comparing…');
  });

  it('Given cards When amounts are masked Then the card link hides them with pointer-only tokens until toggled', async () => {
    render(CreditCardsPage, { data: { engine: never(), banks: settled([testBank()]) } as never });
    await flush();

    const link = screen.getByRole('link', { name: /Blue Rewards/ });
    expect(link.getAttribute('href')).toBe('/ui/accounts/credit-cards/Blue%20Rewards');
    expect(link.className).toContain('interactive');
    expect(link.querySelector('button')).toBeNull();
    expect(link.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/6\.420\.000|50\.000\.000/);

    mask.toggle();
    await flush();
    expect(link.textContent).toMatch(/6\.420\.000/);
  });

  it('Given an open card When the list renders Then its card shows the estimated statement in place of Confirmed', async () => {
    const card = testCard({
      statement: {
        start: '2026-09-21',
        end: '2026-10-20',
        open: false,
        spent: 2_350_000,
        fees: 0,
        cashback: 0,
        amount: 2_350_000,
      },
    });
    render(CreditCardsPage, { data: { engine: never(), banks: settled([testBank([card])]) } as never });
    mask.toggle();
    await flush();

    const link = screen.getByRole('link', { name: /Blue Rewards/ });
    expect(link.textContent).toContain('Est. statement');
    expect(link.textContent).toMatch(/2\.350\.000/);
    expect(link.textContent).not.toContain('Confirmed');
  });
});
