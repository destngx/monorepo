import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Account, AIStatus, CaptureDraft, Category, QuickCaptureOverview } from '$lib/api/types';
import { formatMoney, MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test gives it an answer.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
const answers = vi.hoisted((): Record<string, (...args: unknown[]) => unknown> => ({}));
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          const answer = answers[String(name)];
          return answer ? Promise.resolve(answer(...args)) : new Promise(() => undefined);
        },
    },
  ),
}));
vi.mock('$app/navigation', () => ({
  invalidate: vi.fn(() => Promise.resolve()),
  invalidateAll: vi.fn(() => Promise.resolve()),
}));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import DraftsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};
const pending = () => ({ engine: never(), review: never(), overview: never(), aiStatus: never() });

// Invented data.
const draft: CaptureDraft = {
  id: 'drf_1',
  status: 'pending',
  createdAt: '2026-10-16T09:00:00Z',
  captureIds: ['cap_a'],
  suggestion: {
    id: 's1',
    selected: true,
    transaction: {
      accountName: 'Everyday',
      date: '2026-10-16',
      payee: 'Noodle Bar',
      category: 'Food',
      tags: [],
      cleared: true,
      payment: 85_000,
    },
    direction: 'out',
    amount: 85_000,
    currency: 'VND',
    occurredAt: '2026-10-16 08:15',
    confidence: 0.9,
    lowConfidence: [],
    evidence: [{ materialId: 'cap_a-image' }],
  },
};
const review = {
  drafts: { drafts: [draft], materials: [], balances: [] },
  accounts: [
    { name: 'Everyday', type: 'bank', currency: 'VND', balance: 1_000_000 },
    { name: 'Card Acme', type: 'negative-active', currency: 'VND', balance: -2_000_000 },
    { name: 'Card Duo', type: 'negative-active', currency: 'VND', balance: 0 },
  ] as Account[],
  categories: [
    { name: 'Food', type: 'expense' },
    { name: 'Transport', type: 'expense' },
  ] as Category[],
  tags: [],
  cardTags: new Map([
    ['Card Acme', ['Platinum Online']],
    ['Card Duo', ['Gold', 'Silver']],
  ]),
};
const overview: QuickCaptureOverview = {
  pendingDrafts: 1,
  captures: [
    {
      id: 'cap_a',
      createdAt: '2026-10-16T08:16:00Z',
      status: 'done',
      note: 'team lunch 240.000đ',
      hasImage: true,
      attempts: 1,
      newDrafts: 1,
    },
  ],
};

describe('Drafts page', () => {
  afterEach(() => {
    mask.hide();
    calls.length = 0;
    for (const name of Object.keys(answers)) delete answers[name];
    vi.useRealTimers();
  });

  it('Given nothing has answered When the page renders Then the header and upload show with a skeleton per section', () => {
    render(DraftsPage, { data: pending() as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Drafts' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Choose screenshots' })).toBeTruthy();
    const send = screen.getByRole('button', { name: 'Send' });
    expect(send.hasAttribute('disabled')).toBe(true);
    expect(send.getAttribute('title')).toBe('Choose a screenshot or write a note first');
    expect(screen.getByText('Loading drafts…')).toBeTruthy();
    expect(screen.getByText('Loading captures…')).toBeTruthy();
  });

  it('Given the captures list fails When it settles Then it shows the error with Retry and the drafts keep loading', async () => {
    render(DraftsPage, {
      data: { ...pending(), overview: defer(Promise.reject(new Error('capture inbox offline'))) } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('captures');
    expect(alert.textContent).toContain('capture inbox offline');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('Loading drafts…')).toBeTruthy();
  });

  it('Given AI is off When the status loads Then the page explains it', async () => {
    render(DraftsPage, { data: { ...pending(), aiStatus: settled<AIStatus | null>(null) } as never });
    await flush();

    expect(screen.getByText(/AI is not configured on the engine/)).toBeTruthy();
  });

  it('Given a draft When Approve is clicked Then it stays busy until the sheet answers and Reject waits', async () => {
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    const approve = screen.getByRole('button', { name: 'Approve 1 row' });
    await fireEvent.click(approve);
    await flush();

    expect(calls.some((call) => call.name === 'approveDrafts')).toBe(true);
    expect(approve.getAttribute('aria-busy')).toBe('true');
    expect(approve.textContent).toContain('Writing…');
    expect(screen.getByRole('button', { name: 'Reject 1' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given a capture note quoting an amount When masked Then the amount stays hidden until toggled', async () => {
    render(DraftsPage, { data: { ...pending(), overview: settled(overview) } as never });
    await flush();

    expect(screen.getByText(/team lunch/).textContent).toBe(`“team lunch ${MASK}”`);
    expect(document.body.textContent).not.toContain('240.000');

    mask.toggle();
    await flush();
    expect(document.body.textContent).toContain('240.000đ');
  });

  it('Given a draft When its payee changes Then the category is suggested again for the new payee', async () => {
    vi.useFakeTimers();
    answers.suggestCategory = () => ({ category: 'Transport', source: 'ai' });
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    await fireEvent.input(screen.getByLabelText('Payee'), { target: { value: 'Grab Bike' } });
    await vi.advanceTimersByTimeAsync(1000);
    await flush();

    expect(calls.filter((call) => call.name === 'suggestCategory').map((call) => call.args[0])).toEqual(['Grab Bike']);
    expect(screen.getByLabelText<HTMLSelectElement>('Category').value).toBe('Transport');
    expect(screen.getByText('from ai')).toBeTruthy();
  });

  it('Given a category picked by hand When the payee changes Then it is kept', async () => {
    vi.useFakeTimers();
    answers.suggestCategory = () => ({ category: 'Food', source: 'ai' });
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    await fireEvent.change(screen.getByLabelText('Category'), { target: { value: 'Transport' } });
    await fireEvent.input(screen.getByLabelText('Payee'), { target: { value: 'Grab Bike' } });
    await vi.advanceTimersByTimeAsync(1000);
    await flush();

    expect(calls.some((call) => call.name === 'suggestCategory')).toBe(false);
    expect(screen.getByLabelText<HTMLSelectElement>('Category').value).toBe('Transport');
    expect(screen.getByText('picked by you')).toBeTruthy();
  });

  it('Given an edited draft When Save is clicked Then the edit is stored without approving', async () => {
    answers.editDraft = () => undefined;
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    const save = screen.getByRole('button', { name: 'Save' });
    expect(save.hasAttribute('disabled')).toBe(true);
    await fireEvent.input(screen.getByLabelText('Payee'), { target: { value: 'Noodle House' } });
    expect(screen.getByText('Unsaved changes')).toBeTruthy();
    await fireEvent.click(save);
    await flush();

    const edit = calls.find((call) => call.name === 'editDraft');
    expect(edit?.args[0]).toBe('drf_1');
    expect(edit?.args[1]).toMatchObject({ transaction: { payee: 'Noodle House', category: 'Food' } });
    expect(calls.some((call) => call.name === 'approveDrafts')).toBe(false);
    expect(screen.queryByText('Unsaved changes')).toBeNull();
    expect(save.hasAttribute('disabled')).toBe(true);
  });

  it('Given two drafts When the later one is approved alone Then only that draft is sent', async () => {
    const second: CaptureDraft = {
      ...draft,
      id: 'drf_2',
      createdAt: '2026-10-15T09:00:00Z',
      suggestion: {
        ...draft.suggestion,
        occurredAt: '2026-10-16 19:30',
        transaction: { ...draft.suggestion.transaction, payee: 'Tea Shop' },
      },
    };
    render(DraftsPage, {
      data: {
        ...pending(),
        review: settled({ ...review, drafts: { ...review.drafts, drafts: [draft, second] } }),
      } as never,
    });
    await flush();

    const [, approveSecond] = screen.getAllByRole('button', { name: 'Approve' });
    await fireEvent.click(approveSecond);
    await flush();
    const approval = calls.find((call) => call.name === 'approveDrafts');
    expect((approval?.args[1] as { id: string }[]).map((row) => row.id)).toEqual(['drf_2']);
  });

  it('Given a draft When its own Reject is clicked Then only it is rejected', async () => {
    answers.rejectDrafts = () => undefined;
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Reject' }));
    await flush();

    expect(calls.find((call) => call.name === 'rejectDrafts')?.args[0]).toEqual(['drf_1']);
    expect(screen.queryByLabelText('Payee')).toBeNull();
  });

  it('Given selected drafts When the page renders Then each account shows its balance after approve', async () => {
    mask.toggle();
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    expect(screen.getByText('Balances after approve')).toBeTruthy();
    const after = formatMoney(915_000, 'VND');
    expect(screen.getAllByText(after).length).toBe(2);
    expect(screen.getByText(/Balance after:/).textContent).toContain(after);

    await fireEvent.click(screen.getByRole('checkbox', { name: 'Select Noodle Bar' }));
    expect(screen.queryByText('Balances after approve')).toBeNull();
  });

  it('Given card spending When no card is picked Then approve is refused until one is, and an account with one card fills it', async () => {
    render(DraftsPage, { data: { ...pending(), review: settled(review) } as never });
    await flush();

    await fireEvent.change(screen.getByLabelText('Account'), { target: { value: 'Card Duo' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Approve' }));
    await flush();
    expect(screen.getByText('Pick the card: Gold or Silver.')).toBeTruthy();
    expect(calls.some((call) => call.name === 'approveDrafts')).toBe(false);

    await fireEvent.change(screen.getByLabelText('Card'), { target: { value: 'Silver' } });
    await flush();
    expect(screen.queryByText('Pick the card: Gold or Silver.')).toBeNull();

    await fireEvent.change(screen.getByLabelText('Account'), { target: { value: 'Card Acme' } });
    await flush();
    expect(screen.queryByLabelText('Card')).toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: 'Approve' }));
    await flush();
    const approval = calls.find((call) => call.name === 'approveDrafts');
    expect((approval?.args[1] as { transaction: { tags: string[] } }[])[0].transaction.tags).toEqual([
      'Platinum Online',
    ]);
  });
});
