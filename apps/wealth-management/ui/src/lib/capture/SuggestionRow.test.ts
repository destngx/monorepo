import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import type { CaptureSuggestion } from '$lib/api/types';
import { toReviewRows } from './review';
import SuggestionRow from './SuggestionRow.svelte';

const suggestion = (extra: Partial<CaptureSuggestion>): CaptureSuggestion => ({
  id: 's1',
  selected: false,
  transaction: {
    accountName: 'Vietcombank',
    date: '07/10/2026',
    payee: 'Grab',
    category: 'Transport',
    payment: 120000,
    tags: [],
    cleared: true,
  },
  direction: 'out',
  amount: 120000,
  currency: 'VND',
  occurredAt: '2026-10-07 08:00',
  confidence: 0.9,
  lowConfidence: [],
  evidence: [],
  ...extra,
});

const renderRow = (value: CaptureSuggestion) =>
  render(SuggestionRow, {
    row: toReviewRows({ suggestions: [value], unreadable: [], balances: [] })[0],
    disabled: false,
    problems: [],
    usdPreview: null,
    itemById: new Map(),
    activeAccounts: [],
    categoryNames: [],
    grouped: [],
    tags: [],
    balanceAfter: null,
    balanceCurrency: 'VND',
    cardTags: new Map(),
    verb: 'Approve',
    actionsDisabled: false,
    dirty: false,
    saving: false,
    onapprove: () => Promise.resolve(),
  });

describe('SuggestionRow', () => {
  it('Given a row on or before the last recorded date When shown Then it says it is possibly recorded', () => {
    renderRow(suggestion({ recordedThrough: '08/10/2026' }));

    expect(screen.getByText(/Possibly already recorded: Vietcombank is recorded up to/)).toBeTruthy();
    expect(screen.queryByText(/Likely already recorded/)).toBeNull();
  });

  it('Given a new row When shown Then there is no duplicate warning', () => {
    renderRow(suggestion({ selected: true }));

    expect(screen.queryByText(/already recorded/)).toBeNull();
  });

  it('Given a payee matched to a near name in money lent When shown Then the payee is highlighted to check', () => {
    renderRow(suggestion({ lowConfidence: ['payee'] }));

    expect(screen.getByLabelText('Payee').className).toContain('ring-amber-400/70');
  });

  it('Given a confident payee When shown Then the payee is not highlighted', () => {
    renderRow(suggestion({}));

    expect(screen.getByLabelText('Payee').className).not.toContain('ring-amber-400/70');
  });
});
