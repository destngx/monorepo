import { z } from 'zod';

// ============================================================================
// Loan Schemas
// ============================================================================

export const LoanSchema = z.object({
  name: z.string().min(1),
  monthlyDebt: z.number(),
  monthlyPaid: z.number(),
  monthlyRemaining: z.number(),
  extra: z.string(),
  yearlyDebt: z.number(),
  yearlyPaid: z.number(),
  yearlyRemaining: z.number(),
});

export const CreateLoanSchema = LoanSchema.omit({
  monthlyPaid: true,
  monthlyRemaining: true,
  yearlyPaid: true,
  yearlyRemaining: true,
}).extend({
  monthlyPaid: z.number().optional(),
  monthlyRemaining: z.number().optional(),
  yearlyPaid: z.number().optional(),
  yearlyRemaining: z.number().optional(),
});

export const UpdateLoanSchema = LoanSchema.partial();

// ============================================================================
// Type Exports (for use in TypeScript code)
// ============================================================================

export type Loan = z.infer<typeof LoanSchema>;
export type CreateLoan = z.infer<typeof CreateLoanSchema>;
export type UpdateLoan = z.infer<typeof UpdateLoanSchema>;
