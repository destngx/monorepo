import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import { findCard } from '$lib/cards/cards';
import { testBank } from '$lib/cards/cards.testdata';
import { mask } from '$lib/state/mask.svelte';

// Sections that load on their own (recommendations, AI explain) stay pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import CardPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Credit card page', () => {
  afterEach(() => mask.hide());

  it('Given the card is still loading When the page renders Then the name, back link and AI summary show at once', () => {
    render(CardPage, { data: { engine: never(), name: 'Blue Rewards', detail: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Blue Rewards' })).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Credit cards' }).getAttribute('href')).toBe('/ui/accounts/credit-cards');
    expect(screen.getByRole('combobox', { name: 'AI summary period' })).toBeTruthy();
    expect(screen.getAllByText('Loading the card…').length).toBeGreaterThan(0);
  });

  it('Given the cards fail When they settle Then the error and Retry show under the header', async () => {
    render(CardPage, {
      data: {
        engine: never(),
        name: 'Blue Rewards',
        detail: defer(Promise.reject(new Error('Cards tab is invalid'))),
      } as never,
    });
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Cards tab is invalid');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('heading', { level: 1, name: 'Blue Rewards' })).toBeTruthy();
  });

  it('Given an unknown card name When the cards load Then the page says so in place of the report', async () => {
    render(CardPage, {
      data: { engine: never(), name: 'Nope', detail: settled(findCard([testBank()], 'Nope')) } as never,
    });
    await flush();

    expect(screen.getByText('No credit card named Nope')).toBeTruthy();
  });

  it('Given the card has loaded When amounts are masked Then they stay hidden until toggled', async () => {
    render(CardPage, {
      data: { engine: never(), name: 'Blue Rewards', detail: settled(findCard([testBank()], 'Blue Rewards')) } as never,
    });
    await flush();

    expect(screen.getByText('Harbor Bank · Everyday 5% · Harbor Card · expires 12/29')).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'This period' })).toBeTruthy();
    expect(document.body.textContent).not.toMatch(/6\.420\.000|43\.580\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toMatch(/6\.420\.000/);
  });
});
