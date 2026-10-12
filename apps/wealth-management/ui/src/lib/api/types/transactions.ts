import type { CategoryType } from './accounts';

export interface Transaction {
  id: string;
  accountName: string;
  date: string;
  referenceNumber: string | null;
  payee: string;
  tags: string[];
  memo: string | null;
  category: string;
  categoryType?: CategoryType;
  cleared: boolean;
  payment: number | null;
  deposit: number | null;
  accountBalance: number;
  clearedBalance: number;
  runningBalance: number;
  etag: string;
  issues?: string[];
  /** Day the rules attribute the row to (refund memo month rule); absent on older engines. */
  effectiveDate?: string | null;
}

export interface TransactionInput {
  accountName: string;
  date: string;
  referenceNumber?: string;
  payee: string;
  tags: string[];
  memo?: string;
  category: string;
  cleared: boolean;
  payment?: number;
  deposit?: number;
}

export interface TransactionBatchResult {
  batchId: string;
  created: number;
  failed: number;
  replayed: boolean;
  rows: { index: number; id?: string; error?: string }[];
}
