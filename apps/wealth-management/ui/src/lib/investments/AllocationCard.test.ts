import type { Allocation } from '$lib/api/types';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ savePreferences: vi.fn(), allocation: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn() }));
const browser = vi.hoisted(() => ({ targetAllocation: { value: '' } }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$lib/state/settings.svelte', () => browser);

import AllocationCard from './AllocationCard.svelte';

const allocation = (targets: Allocation['targets']): Allocation => ({
  lines: [
    {
      class: 'cash',
      label: 'Cash and deposits',
      value: 60,
      share: 60,
      targeted: true,
      current: 60,
      target: targets?.cash ?? null,
      drift: targets ? 60 - targets.cash : null,
      status: targets ? 'over' : 'no_target',
    },
    {
      class: 'funds',
      label: 'Fund certificates',
      value: 40,
      share: 40,
      targeted: true,
      current: 40,
      target: targets?.funds ?? null,
      drift: targets ? 40 - targets.funds : null,
      status: targets ? 'under' : 'no_target',
    },
    {
      class: 'crypto',
      label: 'Crypto',
      value: 0,
      share: 0,
      targeted: true,
      current: 0,
      target: targets?.crypto ?? null,
      drift: targets ? -targets.crypto : null,
      status: targets ? 'on_target' : 'no_target',
    },
    {
      class: 'moneyLent',
      label: 'Money lent',
      value: 0,
      share: 0,
      targeted: false,
      current: null,
      target: null,
      drift: null,
      status: 'no_target',
    },
  ],
  total: 100,
  targetedTotal: 100,
  targets,
  driftThreshold: 5,
  rebalance: targets !== null,
  unvalued: 0,
  issues: [],
  generatedAt: '2026-10-17T00:00:00Z',
});

async function settle() {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
}

describe('AllocationCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    browser.targetAllocation.value = '';
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
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

  it('given targets only in this browser, when saving them to the sheet, then the browser copy is dropped', async () => {
    browser.targetAllocation.value = '{"cash":50,"funds":40,"crypto":10}';
    const saved = allocation({ cash: 50, funds: 40, crypto: 10 });
    endpoints.savePreferences.mockResolvedValue({});
    endpoints.allocation.mockResolvedValue(saved);
    const onchange = vi.fn();
    render(AllocationCard, { allocation: allocation(null), onchange });

    await fireEvent.click(screen.getByRole('button', { name: 'Save to your sheet' }));
    await settle();

    expect(endpoints.savePreferences).toHaveBeenCalledWith([
      { key: 'allocation.target.cash', value: 50 },
      { key: 'allocation.target.funds', value: 40 },
      { key: 'allocation.target.crypto', value: 10 },
    ]);
    expect(onchange).toHaveBeenCalledWith(saved);
    expect(browser.targetAllocation.value).toBe('');
  });

  it('given targets in the sheet, then drift shows with status labels and no move is offered', () => {
    browser.targetAllocation.value = '{"cash":50,"funds":40,"crypto":10}';
    render(AllocationCard, { allocation: allocation({ cash: 50, funds: 40, crypto: 10 }), onchange: vi.fn() });

    expect(screen.queryByRole('button', { name: 'Save to your sheet' })).toBeNull();
    expect(screen.getByText('Rebalance suggested')).toBeTruthy();
    expect(screen.getByText('Over target')).toBeTruthy();
    expect(screen.getByText('Not targeted')).toBeTruthy();
    expect(browser.targetAllocation.value).toBe('');
  });

  it('given targets that do not add up, when saving, then nothing is sent', async () => {
    render(AllocationCard, { allocation: allocation(null), onchange: vi.fn() });
    await fireEvent.click(screen.getByRole('button', { name: /Set targets/ }));
    await fireEvent.input(screen.getByLabelText('Cash and deposits (%)'), { target: { value: '70' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    await settle();

    expect(endpoints.savePreferences).not.toHaveBeenCalled();
    expect(toastMock.push).toHaveBeenCalledWith('warning', 'Targets must add up to 100% (now 70%).');
  });

  it('given saved targets, when Clear is pressed, then they are only removed once confirmed', async () => {
    endpoints.savePreferences.mockResolvedValue({});
    endpoints.allocation.mockResolvedValue(allocation(null));
    render(AllocationCard, { allocation: allocation({ cash: 50, funds: 40, crypto: 10 }), onchange: vi.fn() });
    await fireEvent.click(screen.getByRole('button', { name: /Edit targets/ }));
    await fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    expect(endpoints.savePreferences).not.toHaveBeenCalled();

    await fireEvent.click(screen.getByRole('button', { name: 'Clear targets', hidden: true }));
    await settle();
    expect(endpoints.savePreferences).toHaveBeenCalledTimes(1);
    expect(toastMock.push).toHaveBeenCalledWith('success', 'Targets cleared.');
  });
});
