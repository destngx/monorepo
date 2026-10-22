import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Preferences } from '$lib/api/types';

const prefs = (dayToDayFloat: number | null): Preferences => ({
  tabExists: true,
  allocationTargets: null,
  goalPool: { emergencyMonths: 6, spendingBufferMonths: 1 },
  safeToSpend: { notReadyAccounts: [], etag: 'e1' },
  idleCash: { dayToDayFloat, emergencyAccounts: [], accountProducts: { accounts: [], etag: 'e4' }, etag: 'e2' },
  accountMinimums: { accounts: [], etag: 'e3' },
  entries: [],
  issues: [],
});

const api = vi.hoisted(() => ({ preferences: vi.fn(), savePreferences: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api }));

import GoalPoolCard from './GoalPoolCard.svelte';

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('GoalPoolCard', () => {
  afterEach(() => vi.clearAllMocks());

  it('Given no float set When the settings load Then the default is named and a typed float is saved to the sheet', async () => {
    api.preferences.mockResolvedValue(prefs(null));
    api.savePreferences.mockResolvedValue(prefs(12_000_000));
    render(GoalPoolCard);
    await flush();

    expect(screen.getByTestId('day-to-day-float').textContent).toContain('the spending buffer');
    await fireEvent.input(screen.getByLabelText('New day-to-day float in VND'), { target: { value: '12000000' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save float' }));
    await flush();

    expect(api.savePreferences).toHaveBeenCalledWith([{ key: 'idleCash.dayToDayFloat', value: 12_000_000 }]);
    expect(screen.getByTestId('day-to-day-float').textContent).toContain('your setting');
    expect(screen.getByRole('button', { name: 'Use default' })).toBeTruthy();
  });
});
