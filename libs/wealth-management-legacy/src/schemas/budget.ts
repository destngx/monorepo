import { z } from 'zod';

// ============================================================================
// Budget Schemas
// ============================================================================

export const BudgetItemSchema = z.object({
  category: z.string().min(1),
  categoryType: z.enum(['income', 'expense', 'non-budget']).optional(),
  monthlyLimit: z.number().positive(),
  yearlyLimit: z.number().positive(),
  monthlyLimits: z.record(z.string(), z.number()).optional(),
  weeklyLimit: z.number().optional(),
  monthlySpent: z.number(),
  yearlySpent: z.number(),
  weeklySpent: z.number().optional(),
  monthlyRemaining: z.number(),
  yearlyRemaining: z.number(),
  weeklyRemaining: z.number().optional(),
  note: z.string().optional(),
});

export const CreateBudgetSchema = BudgetItemSchema.omit({
  monthlySpent: true,
  yearlySpent: true,
  monthlyRemaining: true,
  yearlyRemaining: true,
}).extend({
  monthlySpent: z.number().optional(),
  yearlySpent: z.number().optional(),
  monthlyRemaining: z.number().optional(),
  yearlyRemaining: z.number().optional(),
});

export const UpdateBudgetSchema = BudgetItemSchema.partial();

// Alternative budget schema for updates
export const BudgetUpdateSchema = z.object({
  category: z.string().min(1),
  monthlyLimit: z.number().min(0).optional(),
  yearlyLimit: z.number().min(0).optional(),
});

// ============================================================================
// Type Exports (for use in TypeScript code)
// ============================================================================

export type BudgetItem = z.infer<typeof BudgetItemSchema>;
export type CreateBudgetItem = z.infer<typeof CreateBudgetSchema>;
export type UpdateBudgetItem = z.infer<typeof UpdateBudgetSchema>;
