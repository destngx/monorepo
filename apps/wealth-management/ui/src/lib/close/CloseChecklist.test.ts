import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import CloseChecklist from './CloseChecklist.svelte';
import { monthlyClose } from './close.testdata';

vi.mock('$app/paths', () => ({ base: '/ui' }));

const row = (container: HTMLElement, key: string) => container.querySelector(`[data-item="${key}"]`) as HTMLElement;

describe('CloseChecklist', () => {
  afterEach(() => mask.hide());

  it('heads with the checks done and shows each state with an icon and a label', () => {
    const { container } = render(CloseChecklist, { close: monthlyClose(), onWriteSnapshot: vi.fn() });

    expect(screen.getByTestId('close-headline').textContent).toBe('2 of 7 checks done for Oct 2026; 5 need you.');
    expect(row(container, 'drafts').querySelector('[data-status="warning"]')?.textContent).toContain('To do');
    expect(row(container, 'categorize').querySelector('[data-status="good"]')?.textContent).toContain('Done');
    expect(row(container, 'goals').querySelector('[data-status="neutral"]')?.textContent).toContain('Unavailable');
    expect(row(container, 'drafts').querySelector('[data-status] svg')).toBeTruthy();
  });

  it('links each item to where the user acts and masks the amounts', () => {
    const { container } = render(CloseChecklist, { close: monthlyClose(), onWriteSnapshot: vi.fn() });

    expect(row(container, 'budget').querySelector('a')?.getAttribute('href')).toBe('/ui/budget?month=2026-10');
    expect(row(container, 'recommendations').querySelector('a')?.getAttribute('href')).toBe('/ui/recommendations');
    expect(row(container, 'budget').textContent).toContain(`Over by ${MASK}`);
    expect(row(container, 'recommendations').textContent).toContain(`Estimated impact a year ${MASK}`);
  });

  it('writes the snapshot from its item and offers an update once written', async () => {
    const write = vi.fn();
    const { container, rerender } = render(CloseChecklist, { close: monthlyClose(), onWriteSnapshot: write });

    await fireEvent.click(screen.getByText('Write the snapshot', { selector: 'button' }));
    expect(write).toHaveBeenCalledOnce();

    mask.toggle();
    await rerender({
      close: monthlyClose({
        snapshot: {
          month: '2026-10',
          asOf: '2026-10-31',
          netWorth: 120_000_000,
          assets: null,
          liabilities: null,
          moneyLent: null,
          cash: null,
          funds: null,
          crypto: null,
          income: null,
          expense: null,
          savings: null,
          savingsRate: 25,
          healthScore: 71.6,
          savingsScore: null,
          emergencyScore: null,
          debtScore: null,
          diversificationScore: null,
          emergencyMonths: null,
          writtenOn: '2026-11-01',
          note: '',
          row: 2,
        },
      }),
      onWriteSnapshot: write,
    });
    expect(screen.getByText('Update the snapshot')).toBeTruthy();
    expect(row(container, 'snapshot').textContent).toContain('120.000.000');
    expect(screen.getByTestId('close-snapshot').textContent).toMatch(/Health score\s*72/);
  });

  it('offers no write for a month too old to snapshot', () => {
    render(CloseChecklist, { close: monthlyClose({ canWriteSnapshot: false }), onWriteSnapshot: vi.fn() });

    expect(screen.queryByText('Write the snapshot', { selector: 'button' })).toBeNull();
  });
});
