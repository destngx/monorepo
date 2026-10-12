import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import Masked from './Masked.svelte';
import Money from './Money.svelte';

afterEach(() => mask.hide());

describe('Money', () => {
  it('is masked by default and reveals every amount on click', async () => {
    render(Money, { value: 1_250_000 });
    render(Masked, { text: 'Lender A' });

    const amount = screen.getByRole('button');
    expect(amount.textContent).toBe(MASK);
    expect(screen.queryByText('Lender A')).toBeNull();

    await fireEvent.click(amount);

    expect(amount.textContent).toBe('1.250.000 ₫');
    expect(screen.getByText('Lender A')).toBeTruthy();
  });

  it('signs and colours by direction', async () => {
    mask.toggle();
    render(Money, { value: 500, signed: true, tone: true });
    const amount = await screen.findByRole('button');
    expect(amount.textContent).toBe('+500 ₫');
    expect(amount.className).toContain('text-emerald-600');
  });

  it('formats foreign currencies with decimals', () => {
    mask.toggle();
    render(Money, { value: 12.5, currency: 'USD' });
    expect(screen.getByRole('button').textContent).toBe('12.50 USD');
  });
});
