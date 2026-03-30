import { z } from 'zod';

// ============================================================================
// Transaction Schemas
// ============================================================================

export const TransactionSchema = z.object({
  id: z.string(),
  accountName: z.string().min(1),
  date: z.coerce.date(),
  referenceNumber: z.string().nullable(),
  payee: z.string().min(1),
  tags: z.array(z.string()),
  memo: z.string().nullable(),
  category: z.string().min(1),
  categoryType: z.enum(['income', 'expense', 'non-budget']).optional(),
  cleared: z.boolean(),
  payment: z.number().nullable(),
  deposit: z.number().nullable(),
  accountBalance: z.number(),
  clearedBalance: z.number(),
  runningBalance: z.number(),
});

export const CreateTransactionSchema = TransactionSchema.omit({
  id: true,
  accountBalance: true,
  clearedBalance: true,
  runningBalance: true,
});

export const UpdateTransactionSchema = TransactionSchema.partial();

// Alternative transaction schema for input validation
export const TransactionInputSchema = z.object({
  id: z.string().optional(),
  accountName: z.string().min(1, 'Account name is required'),
  date: z.date(),
  referenceNumber: z.string().nullable().optional(),
  payee: z.string().min(1, 'Payee is required'),
  tags: z.array(z.string()).default([]),
  memo: z.string().nullable().optional(),
  category: z.string().min(1, 'Category is required'),
  cleared: z.boolean().default(false),
  payment: z.number().nullable().optional(),
  deposit: z.number().nullable().optional(),
});

// ============================================================================
// Type Exports (for use in TypeScript code)
// ============================================================================

export type Transaction = z.infer<typeof TransactionSchema>;
export type CreateTransaction = z.infer<typeof CreateTransactionSchema>;
export type UpdateTransaction = z.infer<typeof UpdateTransactionSchema>;
export type TransactionInput = z.infer<typeof TransactionInputSchema>;
