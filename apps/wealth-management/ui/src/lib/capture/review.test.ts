import { describe, expect, it } from 'vitest';
import type { CaptureResult, CaptureSuggestion } from '$lib/api/types';
import {
  cleanInput,
  commitBatch,
  confidenceLevel,
  learnedRules,
  reconcile,
  rowProblems,
  sheetDateToISO,
  toReviewRows,
  type ReviewRow,
} from './review';

function suggestion(overrides: Partial<CaptureSuggestion> = {}): CaptureSuggestion {
  return {
    id: 's1',
    selected: true,
    transaction: {
      accountName: 'Vietcombank',
      date: '03/10/2026',
      payee: 'Highlands Coffee',
      category: 'Coffee',
      tags: [],
      cleared: true,
      payment: 45000,
      memo: '[capture] Vietcombank text',
    },
    direction: 'out',
    amount: 45000,
    currency: 'VND',
    occurredAt: '2026-10-03 12:30',
    confidence: 0.9,
    lowConfidence: [],
    evidence: [{ materialId: 'm1', quote: '-45,000VND' }],
    ...overrides,
  };
}

const result = (suggestions: CaptureSuggestion[]): CaptureResult => ({ suggestions, unreadable: [], balances: [] });

describe('toReviewRows', () => {
  it('turns suggestions into editable rows with ISO dates', () => {
    const [row] = toReviewRows(result([suggestion({ selected: false })]));
    expect(row.selected).toBe(false);
    expect(row.tx.date).toBe('2026-10-03');
    expect(row.tx.referenceNumber).toBe('');
    expect(sheetDateToISO('2026-01-02')).toBe('2026-01-02');
  });
});

describe('reconcile', () => {
  it('projects the sheet balance with selected and committed rows', () => {
    const rows = toReviewRows(
      result([
        suggestion(),
        suggestion({ id: 's2', transaction: { ...suggestion().transaction, payment: undefined, deposit: 10000 } }),
        suggestion({ id: 's3', selected: false }),
      ]),
    );
    rows[1].selected = false;
    rows[1].createdId = 'row-10';
    const [line] = reconcile(
      [
        {
          account: 'Vietcombank',
          currency: 'VND',
          balanceAfter: 1_000_000,
          asOf: '2026-10-03 12:30',
          materialId: 'm1',
          sheetBalance: 1_030_000,
        },
      ],
      rows,
    );
    expect(line.pending).toBe(-35000);
    expect(line.projected).toBe(995_000);
    expect(line.difference).toBe(5000);
  });
});

describe('rowProblems', () => {
  const accounts = ['Vietcombank'];
  const categories = ['Coffee'];

  it('accepts a complete row', () => {
    expect(rowProblems(toReviewRows(result([suggestion()]))[0].tx, accounts, categories)).toEqual([]);
  });

  it('lists what the engine would reject', () => {
    const tx = {
      ...toReviewRows(result([suggestion()]))[0].tx,
      accountName: '',
      category: 'Uncategorized',
      deposit: 5,
    };
    expect(rowProblems(tx, accounts, categories)).toEqual([
      'Pick an account.',
      'Pick a category.',
      'Enter either a payment or a deposit.',
    ]);
  });
});

describe('commitBatch', () => {
  it('sends selected rows that are not in the sheet yet, cleaned', () => {
    const rows = toReviewRows(
      result([suggestion(), suggestion({ id: 's2' }), suggestion({ id: 's3', selected: false })]),
    );
    rows[0].createdId = 'row-1';
    rows[1].tx.payee = '  Grab  ';
    const batch = commitBatch(rows);
    expect(batch.indices).toEqual([1]);
    expect(batch.inputs[0]).toEqual({
      accountName: 'Vietcombank',
      date: '2026-10-03',
      payee: 'Grab',
      category: 'Coffee',
      tags: [],
      cleared: true,
      memo: '[capture] Vietcombank text',
      payment: 45000,
    });
    expect(cleanInput({ ...rows[1].tx, memo: ' ', referenceNumber: ' FT1 ' }).referenceNumber).toBe('FT1');
  });
});

describe('learnedRules', () => {
  const committed = (row: ReviewRow): ReviewRow => ({ ...row, createdId: 'row-1' });

  it('learns unresolved account hints and guessed or corrected categories', () => {
    const rows = toReviewRows(
      result([
        suggestion({ accountHint: 'VCB *1234', accountSource: 'alias', categorySource: 'history' }),
        suggestion({ id: 's2', accountHint: 'Ví điện tử MoMo', accountSource: 'name', categorySource: 'ai' }),
        suggestion({ id: 's3', accountHint: 'Vietcombank', categorySource: 'history' }),
        suggestion({ id: 's4', transferGroup: 't1', categorySource: 'transfer', accountHint: 'TK 9999' }),
        suggestion({ id: 's5', categorySource: 'default' }),
      ]),
    ).map(committed);
    rows[1].tx.accountName = 'Ví MoMo';
    rows[1].tx.payee = 'Grab';
    rows[1].tx.category = 'Transport';
    rows[2].tx.category = 'Food';
    rows[4].tx.category = 'Uncategorized';
    expect(learnedRules(rows)).toEqual([
      { kind: 'alias', pattern: 'Ví điện tử MoMo', value: 'Ví MoMo' },
      { kind: 'payee', pattern: 'Grab', value: 'Transport' },
      { kind: 'payee', pattern: 'Highlands Coffee', value: 'Food' },
      { kind: 'alias', pattern: 'TK 9999', value: 'Vietcombank' },
    ]);
  });

  it('ignores rows that were not committed', () => {
    expect(learnedRules(toReviewRows(result([suggestion({ categorySource: 'ai' })])))).toEqual([]);
  });
});

describe('confidenceLevel', () => {
  it('uses the engine threshold for low', () => {
    expect(confidenceLevel(0.95)).toBe('high');
    expect(confidenceLevel(0.7)).toBe('medium');
    expect(confidenceLevel(0.69)).toBe('low');
  });
});
