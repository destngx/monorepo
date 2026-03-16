import { z } from 'zod';

// ============================================================================
// Account Schemas
// ============================================================================

export const AccountTypeSchema = z.enum([
  'active use',
  'rarely use',
  'long holding',
  'deprecated',
  'negative active use',
  'bank',
  'crypto',
  'cash',
  'investment',
]);

export const CurrencySchema = z.enum(['VND', 'USD', 'USDT']);

export const AccountSchema = z.object({
  name: z.string().min(1, 'Account name is required'),
  dueDate: z.string().nullable(),
  goalAmount: z.number().nullable(),
  goalProgress: z.number().nullable(),
  clearedBalance: z.number(),
  balance: z.number(),
  type: AccountTypeSchema,
  currency: CurrencySchema,
  note: z.string().nullable(),
});

export const CreateAccountSchema = AccountSchema.omit({
  clearedBalance: true,
  balance: true,
}).extend({
  balance: z.number().optional(),
  clearedBalance: z.number().optional(),
});

export const UpdateAccountSchema = AccountSchema.partial();

// ============================================================================
// Type Exports (for use in TypeScript code)
// ============================================================================

export type Account = z.infer<typeof AccountSchema>;
export type CreateAccount = z.infer<typeof CreateAccountSchema>;
export type UpdateAccount = z.infer<typeof UpdateAccountSchema>;
