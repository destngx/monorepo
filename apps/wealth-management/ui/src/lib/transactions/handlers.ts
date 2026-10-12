import type { Transaction } from '$lib/api/types';
import type { SortKey } from './filter';

// Callback shapes shared by the transactions components.
export type TransactionHandler = (tx: Transaction) => void;
export type SortHandler = (key: SortKey) => void;
export type RowToggleHandler = (id: string, checked: boolean) => void;
export type CheckedHandler = (checked: boolean) => void;
