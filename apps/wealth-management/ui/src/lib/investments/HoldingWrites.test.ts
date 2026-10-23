import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ fill: vi.fn(), recordFundTrade: vi.fn(), setCryptoHoldings: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

import type { ValuedHolding } from '$lib/api/types';
import CryptoHoldingsModal from './CryptoHoldingsModal.svelte';
import FundTradeModal from './FundTradeModal.svelte';

async function describeIt(text: string) {
  await fireEvent.input(screen.getByRole('textbox', { name: 'Describe it' }), { target: { value: text } });
  await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));
}

beforeEach(() => {
  vi.clearAllMocks();
  if (!('showModal' in HTMLDialogElement.prototype)) {
    Object.assign(HTMLDialogElement.prototype, {
      showModal(this: HTMLDialogElement) {
        this.open = true;
      },
      close(this: HTMLDialogElement) {
        this.open = false;
      },
    });
  }
});

describe('FundTradeModal (WM-FILL-10)', () => {
  it('fills a sale from a sentence as sold units and records it with negative units', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'fund-trade',
      values: { code: 'funda', date: '2026-10-01', units: -12.5, amount: 250_000 },
      filled: ['code', 'date', 'units', 'amount'],
      notes: [],
    });
    endpoints.recordFundTrade.mockResolvedValue({
      code: 'FUNDA',
      units: 87.5,
      entry: '1-10-2026,-12,5',
      row: 4,
      created: false,
    });
    const onclose = vi.fn();
    render(FundTradeModal, { open: true, codes: ['FUNDA'], onclose });

    await describeIt('sold 12.5 units of FUNDA on 1 Oct for 250k');

    expect(await screen.findByDisplayValue('FUNDA')).toBeTruthy();
    expect(screen.getByLabelText<HTMLSelectElement>('Trade').value).toBe('sell');
    expect(screen.getByLabelText<HTMLInputElement>('Units', { exact: false }).value).toBe('12.5');
    expect(screen.getAllByTitle('Filled by AI. Check it before saving.')).toHaveLength(4);

    await fireEvent.click(screen.getByRole('button', { name: 'Record' }));
    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.recordFundTrade).toHaveBeenCalledWith({ code: 'FUNDA', date: '2026-10-01', units: -12.5 });
    expect(toastMock.success).toHaveBeenCalledWith('Recorded FUNDA: 87,5 units held now.');
  });
});

describe('CryptoHoldingsModal (WM-FILL-11)', () => {
  const held = { section: 'crypto', platform: 'Exchange A', symbol: 'BTC', quantity: 0.1 } as ValuedHolding;

  it('lists the coins read from a screenshot against the sheet and saves them in one write', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'crypto-holdings',
      values: {},
      rows: [
        { platform: 'Exchange A', symbol: 'btc', quantity: 0.25 },
        { platform: 'Exchange A', symbol: 'SOL', quantity: 10, note: 'staked' },
      ],
      filled: ['rows'],
      notes: [],
    });
    endpoints.setCryptoHoldings.mockResolvedValue({
      holdings: [
        { platform: 'Exchange A', symbol: 'BTC', quantity: 0.25, row: 2, created: false },
        { platform: 'Exchange A', symbol: 'SOL', quantity: 10, row: 3, created: true },
      ],
    });
    const onclose = vi.fn();
    render(CryptoHoldingsModal, { open: true, holdings: [held], onclose });

    await describeIt('0.25 BTC and 10 SOL on Exchange A');

    expect(await screen.findByText('Now 0,1 in the sheet.')).toBeTruthy();
    expect(screen.getByText('Not in the sheet yet: adds a row.')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Save 2 balances' }));
    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.setCryptoHoldings).toHaveBeenCalledWith([
      { platform: 'Exchange A', symbol: 'BTC', quantity: 0.25, note: undefined },
      { platform: 'Exchange A', symbol: 'SOL', quantity: 10, note: 'staked' },
    ]);
    expect(toastMock.success).toHaveBeenCalledWith('Saved 2 balances, 1 new.');
  });
});
