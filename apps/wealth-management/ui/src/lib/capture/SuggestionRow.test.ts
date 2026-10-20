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
  const duplicate = { transactionId: 'row-9', date: '05/10/2026', payee: 'Grab', amount: 120000 };

  it('Given a match above 80% When shown Then it says it is likely recorded with the score and gap', () => {
    renderRow(suggestion({ duplicate: { ...duplicate, score: 85, days: 2, likely: true } }));

    expect(screen.getByText(/Likely already recorded \(85% match\): Grab on/)).toBeTruthy();
    expect(screen.getByText(/2 days apart/)).toBeTruthy();
    expect(screen.getByText(/Unselected by default/)).toBeTruthy();
  });

  it('Given a weak match When shown Then it is a possible match that stays selected', () => {
    renderRow(suggestion({ selected: true, duplicate: { ...duplicate, score: 60, days: 6, likely: false } }));

    expect(screen.getByText(/Possible match \(60%\): Grab on/)).toBeTruthy();
    expect(screen.getByText(/Still selected/)).toBeTruthy();
    expect(screen.queryByText(/Likely already recorded/)).toBeNull();
  });

  it('Given a same-day match When shown Then it says same day', () => {
    renderRow(suggestion({ duplicate: { ...duplicate, score: 90, days: 0, likely: true } }));

    expect(screen.getByText(/same day/)).toBeTruthy();
  });

  it('Given a learned payee When shown Then it says what the material showed and how often', () => {
    renderRow(suggestion({ counterparty: 'NGUYEN VAN A', learnedPayee: { counterparty: 'NGUYEN VAN A', count: 3 } }));

    expect(screen.getByText(/Payee learned: NGUYEN VAN A was written as Grab 3 times before/)).toBeTruthy();
  });

  it('Given a learned payee also written under other names When shown Then the others are listed', () => {
    renderRow(
      suggestion({
        lowConfidence: ['payee'],
        learnedPayee: { counterparty: 'TRAN THI B', count: 1, others: ['Lent to B', 'Fruit stall'] },
      }),
    );

    expect(screen.getByText(/TRAN THI B was written as Grab once before/)).toBeTruthy();
    expect(screen.getByText(/Also written as Lent to B, Fruit stall: check the payee/)).toBeTruthy();
  });

  it('Given a payee from a counterparty rule added in the sheet When shown Then it names the rule', () => {
    renderRow(suggestion({ learnedPayee: { counterparty: 'LE VAN C', count: 0 } }));

    expect(screen.getByText(/Payee learned: LE VAN C is Grab in CaptureRules/)).toBeTruthy();
  });

  it('Given a row on or before the last recorded date with no match When shown Then it may be missing from the sheet', () => {
    renderRow(suggestion({ selected: true, recordedThrough: '08/10/2026' }));

    expect(screen.getByText(/May be missing from the sheet: Vietcombank is recorded up to/)).toBeTruthy();
    expect(screen.queryByText(/already recorded/)).toBeNull();
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
