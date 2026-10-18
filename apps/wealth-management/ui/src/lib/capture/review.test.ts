import { describe, expect, it } from 'vitest';
import type { Account, CaptureResult, CaptureSuggestion, CardBankReport } from '$lib/api/types';
import {
  applyCategorySuggestion,
  cardTagOptions,
  fillCardTag,
  requiredCardTags,
  cleanInput,
  commitBatch,
  confidenceLevel,
  learnedRules,
  isRowDirty,
  projectBalances,
  runningBalances,
  reconcile,
  rowKey,
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
    expect(commitBatch(rows, [0, 2]).indices).toEqual([2]);
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

  it('treats a category re-suggested after a payee edit as guessed', () => {
    const [row] = toReviewRows(result([suggestion({ categorySource: 'history' })])).map(committed);
    row.tx.payee = 'Grab';
    row.categorySource = 'ai';
    expect(learnedRules([row])).toEqual([{ kind: 'payee', pattern: 'Grab', value: 'Coffee' }]);
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

describe('applyCategorySuggestion', () => {
  const categories = ['Coffee', 'Transport', 'Uncategorized'];

  it('replaces the category and its source when the payee changes', () => {
    const [row] = toReviewRows(result([suggestion({ categorySource: 'history' })]));
    expect(applyCategorySuggestion(row, { category: 'Transport', source: 'ai' }, categories)).toBe(true);
    expect([row.tx.category, row.categorySource]).toEqual(['Transport', 'ai']);
  });

  it('keeps a category picked by hand unless asked, and never applies an unknown one', () => {
    const [row] = toReviewRows(result([suggestion()]));
    row.categorySource = 'manual';
    expect(applyCategorySuggestion(row, { category: 'Transport', source: 'ai' }, categories)).toBe(false);
    expect(row.tx.category).toBe('Coffee');
    expect(applyCategorySuggestion(row, { category: 'Transport', source: 'ai' }, categories, { force: true })).toBe(
      true,
    );
    expect(row.tx.category).toBe('Transport');
    expect(applyCategorySuggestion(row, { category: 'Yachts', source: 'ai' }, categories, { force: true })).toBe(false);
  });

  it('leaves transfer rows alone', () => {
    const [row] = toReviewRows(result([suggestion({ transferGroup: 't1', categorySource: 'transfer' })]));
    expect(applyCategorySuggestion(row, { category: 'Transport', source: 'ai' }, categories, { force: true })).toBe(
      false,
    );
  });
});

describe('isRowDirty', () => {
  it('compares the row with the one last saved, ignoring stray spaces', () => {
    const [row] = toReviewRows(result([suggestion()]));
    expect(isRowDirty(row)).toBe(false);
    row.savedKey = rowKey(row);
    row.tx.payee = 'Highlands Coffee ';
    expect(isRowDirty(row)).toBe(false);
    row.tx.category = 'Transport';
    expect(isRowDirty(row)).toBe(true);
  });
});

describe('projectBalances', () => {
  const accounts = [
    { name: 'Vietcombank', balance: 1_000_000, currency: 'VND' },
    { name: 'Card', balance: -200_000, currency: 'VND' },
    { name: 'Idle', balance: 5, currency: 'USD' },
  ] as Account[];

  it('adds the selected rows not yet written to each account they touch', () => {
    const rows = toReviewRows(
      result([
        suggestion(),
        suggestion({ id: 's2', transaction: { ...suggestion().transaction, payment: undefined, deposit: 10_000 } }),
        suggestion({ id: 's3', transaction: { ...suggestion().transaction, accountName: 'Card' } }),
        suggestion({ id: 's4', selected: false, transaction: { ...suggestion().transaction, accountName: 'Idle' } }),
        suggestion({ id: 's5', transaction: { ...suggestion().transaction, accountName: 'Unknown' } }),
      ]),
    );
    rows[2].createdId = 'tx1';

    expect(projectBalances(accounts, rows)).toEqual([
      { account: 'Vietcombank', currency: 'VND', rows: 2, current: 1_000_000, change: -35_000, after: 965_000 },
    ]);
    expect(runningBalances(accounts, rows)).toEqual([955_000, 965_000, null, null, null]);
  });
});

describe('card tags', () => {
  const reports = [
    {
      cards: [
        { accountName: 'Card Acme', keyword: 'Platinum Online' },
        { accountName: 'Card Acme', keyword: 'Old', closedFrom: '2020-01-01' },
        { accountName: 'Card Duo', keyword: 'Gold' },
        { accountName: 'Card Duo', keyword: 'Silver' },
      ],
    },
  ] as unknown as CardBankReport[];
  const cardTags = cardTagOptions(reports);
  const onCard = (account: string, overrides: Partial<CaptureSuggestion> = {}) =>
    toReviewRows(
      result([suggestion({ transaction: { ...suggestion().transaction, accountName: account }, ...overrides })]),
    )[0];

  it('offers each account its open cards only', () => {
    expect([...cardTags]).toEqual([
      ['Card Acme', ['Platinum Online']],
      ['Card Duo', ['Gold', 'Silver']],
    ]);
  });

  it('requires a card tag on card spending, not on transfers, repayments or other accounts', () => {
    const row = onCard('Card Duo');
    const required = requiredCardTags(row, cardTags);
    expect(required).toEqual(['Gold', 'Silver']);
    expect(rowProblems(row.tx, ['Card Duo'], ['Coffee'], required)).toEqual(['Pick the card tag: Gold or Silver.']);
    row.tx.tags = ['silver'];
    expect(rowProblems(row.tx, ['Card Duo'], ['Coffee'], required)).toEqual([]);

    expect(requiredCardTags(onCard('Card Duo', { transferGroup: 't1' }), cardTags)).toEqual([]);
    const repayment = onCard('Card Duo');
    repayment.tx.payment = undefined;
    repayment.tx.deposit = 100;
    expect(requiredCardTags(repayment, cardTags)).toEqual([]);
    expect(requiredCardTags(onCard('Vietcombank'), cardTags)).toEqual([]);
  });

  it('fills the tag of an account with one open card, once', () => {
    const row = onCard('Card Acme');
    fillCardTag(row.tx, cardTags.get('Card Acme')!);
    fillCardTag(row.tx, cardTags.get('Card Acme')!);
    expect(row.tx.tags).toEqual(['Platinum Online']);
    const duo = onCard('Card Duo');
    fillCardTag(duo.tx, cardTags.get('Card Duo')!);
    expect(duo.tx.tags).toEqual([]);
  });
});
