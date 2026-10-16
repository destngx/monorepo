import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import IdleCashCard from './IdleCashCard.svelte';
import { idleCash } from './idleCash.testdata';

const headline = () => screen.getByTestId('idle-headline').textContent?.replace(/\s+/g, ' ') ?? '';

describe('IdleCashCard', () => {
  afterEach(() => mask.hide());

  it('leads with the idle amount, its rate, the best deposit and the extra a year, masked by default', () => {
    const { container } = render(IdleCashCard, { idle: idleCash() });

    expect(headline()).toContain(
      `About ${MASK} sits idle at 0.22%; a deposit at 5.5% (Bank X) would earn about ${MASK} more a year`,
    );
    expect(container.querySelector('[data-status="warning"]')?.textContent).toContain('Worth moving');
  });

  it('shows the amounts once revealed and the math behind them', () => {
    mask.toggle();
    render(IdleCashCard, { idle: idleCash() });

    expect(headline()).toContain('90.000.000');
    expect(screen.getByText('Show the math')).toBeTruthy();
    expect(screen.getByText('Spending buffer (1.5 months of average expense)')).toBeTruthy();
    expect(screen.getByText(/Account A, 0%/)).toBeTruthy();
    expect(screen.queryByText(/Account C/)).toBeNull();
    expect(screen.getByText(/goal money stays pooled/)).toBeTruthy();
  });

  it('says nothing is idle when what stays liquid covers spendable cash', () => {
    const { container } = render(IdleCashCard, {
      idle: idleCash({
        idle: 0,
        verdict: 'nothing_idle',
        reasons: ['What must stay liquid covers all spendable cash.'],
      }),
    });
    expect(headline()).toContain('Nothing sits idle');
    expect(container.querySelector('[data-status="good"]')?.textContent).toContain('Nothing idle');
  });

  it('gives the reasons when moving is not worth it, and handles a rate already above the best', () => {
    render(IdleCashCard, {
      idle: idleCash({ verdict: 'not_worth_it', extraPerYear: 0, currentRate: 6, reasons: ['Only a small gap.'] }),
    });
    expect(headline()).toContain('already at or above the best published deposit rate (5.5%, Bank X)');
    expect(screen.getByText('Only a small gap.')).toBeTruthy();
  });

  it('says when there is no rate to compare with', () => {
    const { container } = render(IdleCashCard, {
      idle: idleCash({ verdict: 'no_rate', best: null, ratesIssue: 'bank rates unavailable', reasons: ['No rate.'] }),
    });
    expect(headline()).toContain('no published deposit rate to compare with');
    expect(container.querySelector('[data-status="neutral"]')?.textContent).toContain('No rate to compare');
  });
});
