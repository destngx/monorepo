import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import type { CaptureSuggestion } from '$lib/api/types';
import { toReviewRows, type ReviewRow } from './review';
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
  overall: 90,
  reasons: [],
  lowConfidence: [],
  evidence: [],
  ...extra,
});

let lastRow: ReviewRow;

const renderRow = (value: CaptureSuggestion) =>
  render(SuggestionRow, {
    ...rowProps(),
    row: (lastRow = toReviewRows({ suggestions: [value], unreadable: [], balances: [] })[0]),
  });

const rowProps = () => ({
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
  cardNames: new Map(),
  verb: 'Approve',
  actionsDisabled: false,
  dirty: false,
  saving: false,
  onapprove: () => Promise.resolve(),
});

describe('SuggestionRow', () => {
  it('Given a draft When shown Then it has one overall confidence and no inline score notes', () => {
    renderRow(
      suggestion({
        overall: 10,
        reasons: [{ text: 'Likely already in the sheet as row-9 (Grab, 05/10/2026), 90% match', effect: -80 }],
        duplicate: {
          transactionId: 'row-9',
          date: '05/10/2026',
          payee: 'Grab',
          amount: 120000,
          score: 90,
          days: 0,
          likely: true,
        },
        recordedThrough: '08/10/2026',
        learnedPayee: { counterparty: 'NGUYEN VAN A', count: 2 },
        notes: ['No time on the material; matched by date only.'],
      }),
    );

    expect(screen.getByRole('button', { name: /Confidence 10%/ })).toBeTruthy();
    expect(screen.queryByText(/Likely already/)).toBeNull();
    expect(screen.queryByText(/May be missing/)).toBeNull();
    expect(screen.queryByText(/Payee learned/)).toBeNull();
    expect(screen.queryByText(/No time on the material/)).toBeNull();
  });

  it('Given a draft When the Note button is clicked Then the note box opens and keeps what is typed', async () => {
    renderRow(suggestion({ counterparty: 'NGUYEN VAN A', pastNotes: ['breakfast stall', 'coffee too'] }));
    expect(screen.queryByLabelText('Note for next time')).toBeNull();
    expect(screen.queryByText(/Your earlier notes/)).toBeNull();

    await fireEvent.click(screen.getByRole('button', { name: 'Add note' }));
    await fireEvent.input(screen.getByLabelText('Note for next time'), {
      target: { value: 'breakfast stall by the office' },
    });

    expect(lastRow.note).toBe('breakfast stall by the office');
    expect(screen.getByText(/Your earlier notes: breakfast stall · coffee too/)).toBeTruthy();
  });

  it('Given a draft with a saved note When shown Then the button says so and the box stays closed', () => {
    const row = toReviewRows({ suggestions: [suggestion({})], unreadable: [], balances: [] })[0];
    row.note = 'saved before';
    render(SuggestionRow, { ...rowProps(), row });

    expect(screen.getByRole('button', { name: 'Edit note' })).toBeTruthy();
    expect(screen.queryByLabelText('Note for next time')).toBeNull();
  });

  it('Given a transfer or fee row When shown Then there is no Note button', () => {
    renderRow(suggestion({ transferGroup: 't1' }));
    expect(screen.queryByRole('button', { name: 'Add note' })).toBeNull();
  });

  it('Given an account with one open card When shown Then the row names the card it uses', () => {
    const row = toReviewRows({
      suggestions: [
        suggestion({
          transaction: { ...suggestion({}).transaction, accountName: 'Credit Card Acme', tags: ['platinum'] },
        }),
      ],
      unreadable: [],
      balances: [],
    })[0];
    render(SuggestionRow, {
      ...rowProps(),
      row,
      cardTags: new Map([['Credit Card Acme', ['platinum']]]),
      cardNames: new Map([['Credit Card Acme', new Map([['platinum', 'Acme Visa Platinum']])]]),
    });

    expect(screen.getByText('Card: Acme Visa Platinum')).toBeTruthy();
    expect(screen.queryByLabelText('Card')).toBeNull();
    expect(screen.getByTitle('platinum: the card that paid, not edited here')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Remove platinum' })).toBeNull();
  });

  it('Given an account with several open cards When shown Then the picker lists them by name', () => {
    const row = toReviewRows({
      suggestions: [suggestion({ transaction: { ...suggestion({}).transaction, accountName: 'Card Duo' } })],
      unreadable: [],
      balances: [],
    })[0];
    render(SuggestionRow, {
      ...rowProps(),
      row,
      cardTags: new Map([['Card Duo', ['Gold', 'Silver']]]),
      cardNames: new Map([
        [
          'Card Duo',
          new Map([
            ['gold', 'Duo Gold'],
            ['silver', 'Duo Silver'],
          ]),
        ],
      ]),
    });

    const picker = screen.getByLabelText<HTMLSelectElement>('Card');
    expect([...picker.options].map((option) => option.textContent)).toEqual([
      'Pick the card…',
      'Duo Gold',
      'Duo Silver',
    ]);
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
