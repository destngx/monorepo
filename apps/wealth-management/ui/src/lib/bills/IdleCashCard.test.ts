import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';

// Writes stay pending unless a test resolves them.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
const answers = vi.hoisted(() => new Map<string, () => Promise<unknown>>());
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          return answers.get(String(name))?.() ?? new Promise(() => undefined);
        },
    },
  ),
}));
const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/paths', () => ({ base: '/ui' }));

import IdleCashCard from './IdleCashCard.svelte';
import { idleCash } from './idleCash.testdata';

const headline = () => screen.getByTestId('idle-headline').textContent?.replace(/\s+/g, ' ') ?? '';
const text = (id: string) => screen.getByTestId(id).textContent?.replace(/\s+/g, ' ') ?? '';
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('IdleCashCard', () => {
  afterEach(() => {
    mask.hide();
    calls.length = 0;
    answers.clear();
    navigation.invalidateAll.mockClear();
  });

  it('Given idle cash worth a 1-month deposit When shown Then it keeps the float ready, names the kind and is masked', () => {
    const { container } = render(IdleCashCard, { idle: idleCash() });

    expect(headline()).toContain(
      `Keep ${MASK} ready in your everyday accounts. The other ${MASK} could earn about ${MASK} a year more in a ` +
        '1-month deposit at Bank X (5.5%, rolls monthly; withdraw early at the no-term rate). It stays reachable.',
    );
    expect(headline()).not.toMatch(/\(\s*\)|,\s*\)/);
    expect(text('idle-ladder')).toContain(`split it into 3 parts of about ${MASK}, opened 10 days apart`);
    expect(container.querySelector('[data-status="warning"]')?.textContent).toContain('Worth moving');
    expect(document.body.textContent).not.toMatch(/90\.000\.000|60\.000\.000|35\.000\.000/);
  });

  it('Given a no-term best When worth moving Then no ladder is suggested', () => {
    const best = { ...idleCash().best!, access: 'no_term' as const, tenor: 'no-term', tenorMonths: 0 };
    render(IdleCashCard, { idle: idleCash({ best, ladder: null }) });
    expect(headline()).toContain('a no-term deposit at Bank X (5.5%, withdraw any time)');
    expect(text('idle-ladder')).toContain('no ladder is needed');
  });

  it('Given emergency accounts When shown Then the coverage and a better instant-access place are given', () => {
    mask.toggle();
    render(IdleCashCard, { idle: idleCash() });

    const emergency = text('idle-emergency');
    expect(emergency).toContain('Target 30.000.000');
    expect(emergency).toContain('hold 20.000.000');
    expect(emergency).toContain('the other 10.000.000 ₫ stays in ready money');
    expect(text('emergency-better-rate')).toContain('earn 4%. A 1-month deposit at Bank X (5.5%');
    expect(text('emergency-better-rate')).toContain('Move it only if the new place is as quick to reach');
    expect(emergency.toLowerCase()).not.toContain('move all');
  });

  it('Given no emergency account When shown Then the whole target is kept from ready money', () => {
    const idle = idleCash();
    render(IdleCashCard, { idle: { ...idle, emergency: { ...idle.emergency, accounts: [], betterRate: false } } });
    expect(text('idle-emergency')).toContain('No emergency account is marked');
    expect(screen.queryByTestId('emergency-better-rate')).toBeNull();
  });

  it('shows the math with the float and the accounts that are never idle once revealed', () => {
    mask.toggle();
    render(IdleCashCard, { idle: idleCash() });

    expect(headline()).toContain('90.000.000');
    expect(screen.getByText('Show the math')).toBeTruthy();
    expect(screen.getByText('Ready money (without emergency accounts)')).toBeTruthy();
    expect(screen.getByText('Day-to-day float (1.5 months of average expense)')).toBeTruthy();
    expect(screen.getByText(/Coin Account, crypto/)).toBeTruthy();
    expect(screen.getByText(/Slow Account, marked not ready to use/)).toBeTruthy();
    expect(screen.getByText(/Account A, 0%/)).toBeTruthy();
    expect(screen.queryByText(/Account C, 1%/)).toBeNull();
    expect(screen.getByText(/goal money stays pooled/)).toBeTruthy();
  });

  it('says nothing is idle when the float and what stays liquid cover ready money', () => {
    const { container } = render(IdleCashCard, {
      idle: idleCash({ idle: 0, verdict: 'nothing_idle', reasons: ['Nothing is idle.'], ladder: null }),
    });
    expect(headline()).toContain('Nothing sits idle: your day-to-day float');
    expect(container.querySelector('[data-status="good"]')?.textContent).toContain('Nothing idle');
  });

  it('gives the reasons when moving is not worth it, and handles a rate already above the best', () => {
    render(IdleCashCard, {
      idle: idleCash({ verdict: 'not_worth_it', extraPerYear: 0, currentRate: 6, reasons: ['Only a small gap.'] }),
    });
    expect(headline()).toContain(
      'already at or above the best instant-access rate (5.5%, a 1-month deposit at Bank X)',
    );
    expect(screen.getByText('Only a small gap.')).toBeTruthy();
    expect(screen.queryByTestId('idle-ladder')).toBeNull();
  });

  it('says when there is no instant-access rate to compare with', () => {
    const { container } = render(IdleCashCard, {
      idle: idleCash({
        verdict: 'no_rate',
        best: null,
        ladder: null,
        ratesIssue: 'no no-term rate',
        reasons: ['No rate.'],
      }),
    });
    expect(headline()).toContain('there is no instant-access rate (no-term or 1-month deposit) to compare with');
    expect(container.querySelector('[data-status="neutral"]')?.textContent).toContain('No rate to compare');
  });

  it('Given an everyday account When its emergency switch is clicked Then the change is sent with the ETag', async () => {
    render(IdleCashCard, { idle: idleCash() });

    const everyday = screen.getByRole('switch', { name: 'Account A is an everyday account' });
    await fireEvent.click(everyday);
    await flush();

    expect(calls.find((call) => call.name === 'setEmergencyAccount')?.args).toEqual([
      { account: 'Account A', emergency: true, etag: 'etag-1' },
    ]);
    expect(everyday.getAttribute('aria-busy')).toBe('true');
    expect(
      screen.getByRole('switch', { name: 'Rainy Day Account holds the emergency fund' }).hasAttribute('disabled'),
    ).toBe(true);
  });

  it('Given an emergency account When its switch is clicked Then it is unmarked and the page reloads', async () => {
    answers.set('setEmergencyAccount', () => Promise.resolve({}));
    render(IdleCashCard, { idle: idleCash() });

    await fireEvent.click(screen.getByRole('switch', { name: 'Rainy Day Account holds the emergency fund' }));
    await flush();
    expect(calls.find((call) => call.name === 'setEmergencyAccount')?.args).toEqual([
      { account: 'Rainy Day Account', emergency: false, etag: 'etag-1' },
    ]);
    expect(navigation.invalidateAll).toHaveBeenCalled();
  });

  it('Given a typed float When saved Then the preference is written and the page reloads; the input starts empty', async () => {
    answers.set('savePreferences', () => Promise.resolve({}));
    render(IdleCashCard, { idle: idleCash() });

    const input = screen.getByLabelText<HTMLInputElement>('New day-to-day float in VND');
    expect(input.value).toBe('');
    expect(screen.getByRole('button', { name: 'Save float' }).hasAttribute('disabled')).toBe(true);
    await fireEvent.input(input, { target: { value: '20.000.000' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save float' }));
    await flush();

    expect(calls.find((call) => call.name === 'savePreferences')?.args).toEqual([
      [{ key: 'idleCash.dayToDayFloat', value: 20_000_000 }],
    ]);
    expect(navigation.invalidateAll).toHaveBeenCalled();
    expect(screen.queryByRole('button', { name: 'Use default' })).toBeNull();
  });

  it('Given a saved float When Use default is clicked Then the preference is cleared', async () => {
    const idle = idleCash();
    idle.components = { ...idle.components, floatFromPreference: true };
    idle.emergency = { ...idle.emergency, settings: { ...idle.emergency.settings, dayToDayFloat: 35_000_000 } };
    render(IdleCashCard, { idle });

    expect(text('day-to-day-float')).toContain('your setting');
    await fireEvent.click(screen.getByRole('button', { name: 'Use default' }));
    await flush();
    expect(calls.find((call) => call.name === 'savePreferences')?.args).toEqual([
      [{ key: 'idleCash.dayToDayFloat', value: null }],
    ]);
  });

  it('Given a bad float When typed Then it says why and cannot be saved', async () => {
    render(IdleCashCard, { idle: idleCash() });
    const input = screen.getByLabelText('New day-to-day float in VND');
    await fireEvent.input(input, { target: { value: 'abc' } });
    expect(screen.getByRole('alert').textContent).toContain('whole VND');
    expect(screen.getByRole('button', { name: 'Save float' }).hasAttribute('disabled')).toBe(true);
  });
});
