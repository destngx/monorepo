import { describe, it, expect } from 'vitest';
import { mapAccount, mapTransaction, mapBudgetItem } from './mappers';

describe('Sheet Mappers', () => {
  describe('mapAccount', () => {
    it('maps a full valid row correctly with raw VND values', () => {
      // With UNFORMATTED_VALUE, sheet returns raw numbers as strings
      const row = ['Vietcombank', '', '', '', '1000000', '1500000', 'active use', 'My main account', ''];
      const account = mapAccount(row);

      expect(account).not.toBeNull();
      expect(account!.name).toBe('Vietcombank');
      expect(account!.clearedBalance).toBe(1000000);
      expect(account!.balance).toBe(1500000);
      expect(account!.type).toBe('active use');
      expect(account!.currency).toBe('VND');
      expect(account!.note).toBe('My main account');
    });

    it('all accounts are VND since sheet formulas convert', () => {
      const row = ['Binance', '', '200', '30', '60000', '60000', 'rarely use', '', ''];
      const account = mapAccount(row);

      expect(account).not.toBeNull();
      expect(account!.currency).toBe('VND');
      expect(account!.balance).toBe(60000);
    });

    it('handles empty strings and missing indices gracefully', () => {
      const row = ['Cash'];
      const account = mapAccount(row);

      expect(account).not.toBeNull();
      expect(account!.name).toBe('Cash');
      expect(account!.dueDate).toBeNull();
      expect(account!.goalAmount).toBeNull();
      expect(account!.clearedBalance).toBe(0);
      expect(account!.balance).toBe(0);
    });

    it('filters out HELP header rows', () => {
      expect(mapAccount(['HELP', '', '', '', '', '', '', ' See Help'])).toBeNull();
    });

    it('filters out instructional text rows', () => {
      expect(mapAccount(['You can track any number of real or virtual accounts.'])).toBeNull();
      expect(mapAccount(['Accounts column in the Transactions worksheet.'])).toBeNull();
    });

    it('filters out the ACCOUNTS header row', () => {
      const row = ['ACCOUNTS', 'Date to pay', '', '', '', '', 'Type', 'NOTE', 'Categories:'];
      expect(mapAccount(row)).toBeNull();
    });
  });

  describe('mapTransaction', () => {
    it('maps a full valid transaction with raw values', () => {
      const row = [
        'Vietcombank',
        '45658',
        'REF123',
        'Grocery',
        'food, household',
        'Weekly shopping',
        'Groceries',
        '✓',
        '150000',
        '',
        '500000',
        '500000',
        '350000',
      ];
      const tx = mapTransaction(row, 5);

      expect(tx).not.toBeNull();
      expect(tx!.id).toBe('row-5');
      expect(tx!.accountName).toBe('Vietcombank');
      expect(tx!.payee).toBe('Grocery');
      expect(tx!.payment).toBe(150000);
      expect(tx!.deposit).toBeNull();
      expect(tx!.cleared).toBe(true);
    });

    it('handles various cleared statuses correctly', () => {
      expect(mapTransaction(['Cash', '', '', 'P', '', '', 'Cat', 'y', '1000', '', '', '', ''], 1)!.cleared).toBe(true);
      expect(mapTransaction(['Cash', '', '', 'P', '', '', 'Cat', 'Yes', '1000', '', '', '', ''], 1)!.cleared).toBe(
        true,
      );
      expect(mapTransaction(['Cash', '', '', 'P', '', '', 'Cat', '✓', '1000', '', '', '', ''], 1)!.cleared).toBe(true);
      expect(mapTransaction(['Cash', '', '', 'P', '', '', 'Cat', 'n', '1000', '', '', '', ''], 1)!.cleared).toBe(false);
    });

    it('filters out HELP and header rows', () => {
      expect(mapTransaction(['HELP', '', '', 'To get started...'], 0)).toBeNull();
      expect(mapTransaction(['Account', '', 'Num', 'Payee', 'Tag', 'Memo', 'Category'], 1)).toBeNull();
    });

    it('falls back to Uncategorized for missing category', () => {
      const tx = mapTransaction(['Cash', '45658', '', 'Store', '', '', '', '', '10000', '', '', '', ''], 1);
      expect(tx).not.toBeNull();
      expect(tx!.category).toBe('Uncategorized');
    });
  });

  describe('mapBudgetItem', () => {
    it('parses raw VND budget values correctly', () => {
      const row = ['Vacation', '5000000', '60000000', '1000000', '12000000', '4000000', '48000000'];
      const budget = mapBudgetItem(row);

      expect(budget).not.toBeNull();
      expect(budget!.category).toBe('Vacation');
      expect(budget!.monthlyLimit).toBe(5000000);
      expect(budget!.yearlyLimit).toBe(60000000);
      expect(budget!.monthlySpent).toBe(1000000);
    });

    it('filters out HELP and section headers', () => {
      expect(mapBudgetItem(['HELP'])).toBeNull();
      expect(mapBudgetItem(['INCOME CATEGORIES'])).toBeNull();
      expect(mapBudgetItem(['EXPENSE CATEGORIES'])).toBeNull();
    });

    it('keeps valid single-word categories', () => {
      const row = ['Snacks', '350000', '4200000', '100000', '1200000', '250000', '3000000'];
      const budget = mapBudgetItem(row);
      expect(budget).not.toBeNull();
      expect(budget!.category).toBe('Snacks');
    });

    it('applies zeroes for blanks', () => {
      const budget = mapBudgetItem(['Savings', '', '', '', '', '', '']);
      expect(budget).not.toBeNull();
      expect(budget!.monthlyLimit).toBe(0);
      expect(budget!.yearlyRemaining).toBe(0);
    });
  });
});
