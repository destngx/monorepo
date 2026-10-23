import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({
  fill: vi.fn(),
  cardPrograms: vi.fn(),
  accounts: vi.fn(),
  createCard: vi.fn(),
  updateCard: vi.fn(),
  createCardProgram: vi.fn(),
  saveMerchant: vi.fn(),
}));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(), goto: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$app/navigation', () => navigation);

import type { CardConfig } from '$lib/api/types';
import CardFormModal from './CardFormModal.svelte';
import MerchantFormModal from './MerchantFormModal.svelte';
import ProgramFormModal from './ProgramFormModal.svelte';

const config: CardConfig = {
  programs: [{ name: 'Gold', bank: 'Acme', versions: [] }],
  banks: [{ name: 'Acme', sharedAccount: 'Credit Card Acme', referenceLimit: 0 }],
  cards: [
    {
      name: 'Gold Card',
      program: 'Gold',
      keyword: 'gold',
      expiry: '08/2029',
      statementDay: 5,
      dueDay: 25,
      last4: '0123',
    },
  ],
  merchants: [],
};

async function describeIt(text: string) {
  await fireEvent.input(screen.getByRole('textbox', { name: 'Describe it' }), { target: { value: text } });
  await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));
}

beforeEach(() => {
  vi.clearAllMocks();
  endpoints.cardPrograms.mockResolvedValue(config);
  endpoints.accounts.mockResolvedValue([{ name: 'Cash' }, { name: 'Credit Card Beta' }]);
  if (!('showModal' in HTMLDialogElement.prototype)) {
    Object.assign(HTMLDialogElement.prototype, {
      showModal(this: HTMLDialogElement) {
        this.open = true;
      },
      close(this: HTMLDialogElement) {
        this.open = false;
      },
    });
  }
});

describe('CardFormModal (WM-FILL-12)', () => {
  it('fills a new card from a statement and adds it with the shared limit', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'card',
      values: { name: 'Gold Plus', program: 'Gold', statementDay: 20, dueDay: 5, limit: 50_000_000, last4: 4567 },
      filled: ['name', 'program', 'statementDay', 'dueDay', 'limit', 'last4'],
      notes: [],
    });
    endpoints.createCard.mockResolvedValue({ name: 'Gold Plus' });
    const onclose = vi.fn();
    render(CardFormModal, { open: true, onclose });

    await describeIt('statement on the 20th, pay by the 5th, limit 50M');
    expect(await screen.findByDisplayValue('Gold Plus')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.createCard).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Gold Plus',
        program: 'Gold',
        statementDay: 20,
        dueDay: 5,
        last4: '4567',
        limit: 50_000_000,
      }),
    );
  });

  it('edits a held card from its saved row and leaves the limit when blank', async () => {
    endpoints.updateCard.mockResolvedValue({ name: 'Gold Card' });
    const onclose = vi.fn();
    render(CardFormModal, { open: true, editing: 'Gold Card', onclose });

    expect(await screen.findByDisplayValue('08/2029')).toBeTruthy();
    await fireEvent.input(screen.getByLabelText('Due day', { exact: false }), { target: { value: '28' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.updateCard).toHaveBeenCalledWith('Gold Card', {
      name: 'Gold Card',
      program: 'Gold',
      keyword: 'gold',
      expiry: '08/2029',
      statementDay: 5,
      dueDay: 28,
      default: undefined,
      closedFrom: undefined,
      last4: '0123',
      limit: undefined,
    });
    expect(navigation.goto).not.toHaveBeenCalled();
  });
});

describe('ProgramFormModal (WM-FILL-13)', () => {
  it('reads the terms into rules, asks for the shared account of a new bank, and saves them', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'cashback-program',
      values: {
        program: 'Blue',
        bank: 'Beta',
        sharedAccount: 'Credit Card Beta',
        from: '2026-11-01',
        capPeriod: 'month',
      },
      rows: [
        { rule: 'Supermarkets', mccs: '5411', rate: 5, cap: 300_000 },
        { rule: 'Others', rate: 0.5, fallback: true },
      ],
      filled: ['program', 'bank', 'sharedAccount', 'from', 'capPeriod', 'rows'],
      notes: [],
    });
    endpoints.createCardProgram.mockResolvedValue({ name: 'Blue', bank: 'Beta', versions: [] });
    const onclose = vi.fn();
    render(ProgramFormModal, { open: true, onclose });

    await describeIt('5% at supermarkets capped at 300k a month, 0.5% on the rest');
    expect(await screen.findByDisplayValue('Credit Card Beta')).toBeTruthy();
    expect(screen.getByText('Beta is a new bank: the account its cards bill to. A card needs it.')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Save terms' }));

    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.createCardProgram).toHaveBeenCalledWith({
      program: 'Blue',
      bank: 'Beta',
      sharedAccount: 'Credit Card Beta',
      from: '2026-11-01',
      annualFee: undefined,
      cap: undefined,
      capPeriod: 'month',
      rules: [
        {
          rule: 'Supermarkets',
          mccs: '5411',
          rate: 5,
          cap: 300_000,
          minTransaction: undefined,
          minCycleSpend: undefined,
          fallback: undefined,
          exclude: undefined,
          capGroup: undefined,
        },
        {
          rule: 'Others',
          mccs: undefined,
          rate: 0.5,
          cap: undefined,
          minTransaction: undefined,
          minCycleSpend: undefined,
          fallback: true,
          exclude: undefined,
          capGroup: undefined,
        },
      ],
    });
  });
});

describe('MerchantFormModal (WM-FILL-14)', () => {
  it('starts from the given payee and saves the MCC read from a receipt', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'merchant',
      values: { mcc: 5411, confidence: 'confirmed', note: 'receipt' },
      filled: ['mcc', 'confidence', 'note'],
      notes: [],
    });
    endpoints.saveMerchant.mockResolvedValue({ pattern: 'Corner Shop', mcc: 5411, confidence: 'confirmed' });
    const onclose = vi.fn();
    render(MerchantFormModal, { open: true, initial: { pattern: 'Corner Shop', mcc: null }, onclose });

    expect(screen.getByDisplayValue('Corner Shop')).toBeTruthy();
    await describeIt('the receipt says MCC 5411');
    expect(await screen.findByDisplayValue('5411')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
    expect(endpoints.saveMerchant).toHaveBeenCalledWith({
      pattern: 'Corner Shop',
      mcc: 5411,
      from: undefined,
      confidence: 'confirmed',
      note: 'receipt',
    });
    expect(toastMock.success).toHaveBeenCalledWith('Saved MCC 5411 for Corner Shop.');
  });
});
