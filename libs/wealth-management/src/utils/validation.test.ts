import { describe, it, expect } from 'vitest';
import { TransactionSchema, BudgetUpdateSchema } from './validation';

describe('Validation Schemas', () => {
  describe('TransactionSchema', () => {
    it('validates a correct transaction payload', () => {
      const payload = {
        accountName: 'Checking',
        date: new Date(),
        payee: 'Grocery Store',
        category: 'Food',
        payment: 50.0,
      };

      const result = TransactionSchema.safeParse(payload);
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.accountName).toBe('Checking');
        expect(result.data.payment).toBe(50.0);
        expect(result.data.cleared).toBe(false); // default value
      }
    });

    it('fails when required fields are missing', () => {
      const payload = {
        date: new Date(),
        category: 'Food',
      };

      const result = TransactionSchema.safeParse(payload);
      expect(result.success).toBe(false);
    });

    it('fails when string constraints are violated (e.g. empty string)', () => {
      const payload = {
        accountName: '',
        date: new Date(),
        payee: 'Grocery Store',
        category: 'Food',
        payment: 50.0,
      };

      const result = TransactionSchema.safeParse(payload);
      expect(result.success).toBe(false);
    });
  });

  describe('BudgetUpdateSchema', () => {
    it('validates correct budget payload', () => {
      const payload = {
        category: 'Food',
        monthlyLimit: 500,
      };
      const result = BudgetUpdateSchema.safeParse(payload);
      expect(result.success).toBe(true);
    });

    it('fails on negative budget limits', () => {
      const payload = {
        category: 'Food',
        monthlyLimit: -100,
      };
      const result = BudgetUpdateSchema.safeParse(payload);
      expect(result.success).toBe(false);
    });
  });
});
