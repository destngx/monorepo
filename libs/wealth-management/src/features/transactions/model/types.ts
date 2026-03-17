export type TransactionType = 'income' | 'expense' | 'non-budget';

export interface Transaction {
  id: string; // Row index as ID
  accountName: string;
  date: Date;
  referenceNumber: string | null;
  payee: string;
  tags: string[];
  memo: string | null;
  category: string;
  categoryType?: 'income' | 'expense' | 'non-budget';
  cleared: boolean;
  payment: number | null; // Debit
  deposit: number | null; // Credit
  accountBalance: number;
  clearedBalance: number;
  runningBalance: number;
}
