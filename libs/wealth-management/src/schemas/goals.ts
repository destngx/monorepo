import { z } from 'zod';

// ============================================================================
// Goal Schemas
// ============================================================================

export const GoalTypeSchema = z.enum([
  'SAVINGS_TARGET',
  'PURCHASE_GOAL',
  'DEBT_PAYOFF',
  'INVESTMENT_TARGET',
  'INCOME_GOAL',
  'NET_WORTH_MILESTONE',
]);

export const AIStatusSchema = z.enum(['ON_TRACK', 'AT_RISK', 'OFF_TRACK']);

export const ContributionSchema = z.object({
  id: z.string(),
  date: z.string(),
  amount: z.number(),
  sourceAccount: z.string(),
  note: z.string().optional(),
});

export const MilestoneSchema = z.object({
  percentage: z.number().min(0).max(100),
  label: z.string(),
  achievedAt: z.string().optional(),
});

export const GoalSchema = z.object({
  id: z.string(),
  name: z.string().min(1),
  type: GoalTypeSchema,
  emoji: z.string(),
  targetAmount: z.number().positive(),
  currentAmount: z.number(),
  deadline: z.string(),
  status: AIStatusSchema,
  linkedAccountId: z.string(),
  contributionType: z.enum(['MANUAL', 'AUTOMATIC', 'AI_MANAGED']),
  streakCount: z.number(),
  milestones: z.array(MilestoneSchema),
  history: z.array(ContributionSchema),
});

export const CreateGoalSchema = GoalSchema.omit({
  id: true,
  status: true,
  streakCount: true,
  milestones: true,
  history: true,
}).extend({
  status: AIStatusSchema.optional(),
  streakCount: z.number().optional(),
  milestones: z.array(MilestoneSchema).optional(),
  history: z.array(ContributionSchema).optional(),
});

export const UpdateGoalSchema = GoalSchema.partial();

// ============================================================================
// Type Exports (for use in TypeScript code)
// ============================================================================

export type Goal = z.infer<typeof GoalSchema>;
export type CreateGoal = z.infer<typeof CreateGoalSchema>;
export type UpdateGoal = z.infer<typeof UpdateGoalSchema>;
export type Contribution = z.infer<typeof ContributionSchema>;
export type Milestone = z.infer<typeof MilestoneSchema>;
