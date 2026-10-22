import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DEFAULT_SWITCH_INPUT, SWITCH_STORAGE_KEY, parseSwitchInput } from '$lib/investments/stablecoinSwitch';
import { mask } from '$lib/state/mask.svelte';

import StablecoinPage from './+page.svelte';

const verdict = () => screen.getByRole('heading', { level: 2, name: /^(Switch|Stay)$/ });
const field = (name: RegExp) => screen.getByRole<HTMLInputElement>('spinbutton', { name });
const setValue = async (input: HTMLInputElement, value: string) => {
  await fireEvent.input(input, { target: { value } });
  await tick();
};

describe('Stablecoin Switch page (WM-INV-30)', () => {
  let saved: Record<string, string>;
  beforeEach(() => {
    saved = {};
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => saved[key] ?? null,
      setItem: (key: string, value: string) => (saved[key] = value),
    });
    mask.revealed = true;
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    mask.hide();
  });

  it('Given nothing saved When the page opens Then it shows the example figures and their verdict at once', () => {
    render(StablecoinPage);

    expect(screen.getByRole('heading', { level: 1, name: 'Stablecoin Switch' })).toBeTruthy();
    expect(field(/Amount/).value).toBe(String(DEFAULT_SWITCH_INPUT.principal));
    // 1000 at 5% vs 10% for 30 days, 0.1% each way.
    expect(verdict().textContent?.trim()).toBe('Switch');
    expect(screen.getByText(/hold at least 15 days/)).toBeTruthy();
    expect(screen.getByText('+2.09 USD')).toBeTruthy();
  });

  it('Given a short hold When the days change Then the verdict turns to Stay with the loss', async () => {
    render(StablecoinPage);

    await setValue(field(/Days held/), '5');

    expect(verdict().textContent?.trim()).toBe('Stay');
    expect(screen.getByText(/fees cost more than the higher rate earns/)).toBeTruthy();
  });

  it('Given a flat entry fee When the fee kind changes Then the fee comes off as an amount', async () => {
    render(StablecoinPage);

    await fireEvent.change(screen.getByRole('combobox', { name: 'Entry fee as' }), { target: { value: 'amount' } });
    await setValue(field(/Entry fee/), '5');

    // 995 x (1 + 10% x 30/365) x 0.999 = 1002.17 vs 1004.11 staying.
    expect(verdict().textContent?.trim()).toBe('Stay');
    expect(screen.getByText('-5.00 USD')).toBeTruthy();
  });

  it('Given a bad figure When typed Then the field is flagged and no verdict is shown', async () => {
    render(StablecoinPage);

    await setValue(field(/Amount/), '');

    expect(field(/Amount/).getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByText('Enter an amount above 0.')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Check your figures' })).toBeTruthy();
    expect(screen.queryByRole('heading', { level: 2, name: /^(Switch|Stay)$/ })).toBeNull();
  });

  it('Given figures typed before When the page opens again Then they are restored, and Reset brings back the example', async () => {
    const first = render(StablecoinPage);
    await setValue(field(/Amount/), '2500');
    expect(parseSwitchInput(saved[SWITCH_STORAGE_KEY]).principal).toBe(2500);
    first.unmount();

    render(StablecoinPage);
    expect(field(/Amount/).value).toBe('2500');

    await fireEvent.click(screen.getByRole('button', { name: 'Reset to example' }));
    await tick();
    expect(field(/Amount/).value).toBe(String(DEFAULT_SWITCH_INPUT.principal));
  });
});
