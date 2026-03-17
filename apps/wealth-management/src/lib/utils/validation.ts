import { z } from "zod";

export const TransactionSchema = z.object({
  id: z.string().optional(),
  accountName: z.string().min(1, "Account name is required"),
  date: z.date(),
  referenceNumber: z.string().nullable().optional(),
  payee: z.string().min(1, "Payee is required"),
  tags: z.array(z.string()).default([]),
  memo: z.string().nullable().optional(),
  category: z.string().min(1, "Category is required"),
  cleared: z.boolean().default(false),
  payment: z.number().nullable().optional(),
  deposit: z.number().nullable().optional(),
});

export type TransactionInput = z.infer<typeof TransactionSchema>;

export const BudgetUpdateSchema = z.object({
  category: z.string().min(1),
  monthlyLimit: z.number().min(0).optional(),
  yearlyLimit: z.number().min(0).optional(),
});
