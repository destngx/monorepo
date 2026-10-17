import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Account, AIStatus, CaptureDraft, Category, QuickCaptureOverview } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          return new Promise(() => undefined);
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
  accounts: [{ name: 'Everyday', type: 'bank', currency: 'VND' } as Account],
  categories: [{ name: 'Food', type: 'expense' }] as Category[],
  tags: [],
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
});
