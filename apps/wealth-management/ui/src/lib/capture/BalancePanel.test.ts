import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { formatMoney } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import BalancePanel from './BalancePanel.svelte';

describe('BalancePanel', () => {
  afterEach(() => mask.hide());

  it('Given accounts with drafts When shown Then each has its balance now and after the selected rows, with the screenshot check', () => {
    render(BalancePanel, {
      verb: 'Approve',
      projections: [
        { account: 'Vietcombank', currency: 'VND', rows: 2, current: 1_000_000, change: -40_000, after: 960_000 },
        { account: 'Ví MoMo', currency: 'VND', rows: 0, current: 50_000, change: 0, after: 50_000 },
      ],
      reconciliation: [
        {
          account: 'Vietcombank',
          currency: 'VND',
          balanceAfter: 960_000,
          asOf: '2026-10-16 08:15',
          materialId: 'cap_1-image',
          sheetBalance: 1_000_000,
          pending: -40_000,
          projected: 960_000,
          difference: 0,
        },
        {
          account: 'Ví MoMo',
          currency: 'VND',
          balanceAfter: 30_000,
          asOf: '2026-10-16 09:00',
          materialId: 'cap_2-image',
          sheetBalance: 50_000,
          pending: 0,
          projected: 50_000,
          difference: -20_000,
        },
      ],
    });

    expect(screen.getByText('Vietcombank')).toBeTruthy();
    expect(screen.getByText('2 selected')).toBeTruthy();
    expect(screen.getByText('None selected')).toBeTruthy();
    expect(screen.getByText('Matches')).toBeTruthy();
    expect(screen.getByText(/Off by/)).toBeTruthy();
  });

  it('Given a card account with a limit When shown Then the credit left leads, with its balance below', () => {
    mask.toggle();
    render(BalancePanel, {
      verb: 'Approve',
      projections: [
        {
          account: 'Card Acme',
          currency: 'VND',
          rows: 1,
          current: -300_000,
          change: -100_000,
          after: -400_000,
          credit: { now: 4_700_000, after: 4_600_000 },
        },
      ],
      reconciliation: [],
    });

    const tile = screen.getByText('Card Acme').closest('li')!;
    expect(tile.textContent).toContain('credit left');
    expect(tile.textContent).toContain(formatMoney(4_600_000, 'VND'));
    expect(tile.textContent).toContain(`Balance ${formatMoney(-400_000, 'VND')}`);
  });
});
