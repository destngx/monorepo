import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@wealth-management/utils";

const categoryBadgeVariants = cva(
  "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium border transition-colors",
  {
    variants: {
      type: {
        income: "bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400 dark:border-emerald-800",
        expense: "bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/30 dark:text-orange-400 dark:border-orange-800",
        'non-budget': "bg-indigo-100 text-indigo-800 border-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400 dark:border-indigo-800",
        'uncategorized': "bg-slate-100 text-slate-800 border-slate-200 dark:bg-slate-900/30 dark:text-slate-400 dark:border-slate-800",
      },
      active: {
        true: "",
        false: "",
      }
    },
    compoundVariants: [
      { type: "income", active: true, class: "bg-emerald-600 text-white border-emerald-600 shadow-sm" },
      { type: "expense", active: true, class: "bg-orange-600 text-white border-orange-600 shadow-sm" },
      { type: "non-budget", active: true, class: "bg-indigo-600 text-white border-indigo-600 shadow-sm" },
    ],
    defaultVariants: {
      type: "expense",
      active: false,
    },
  }
);

export interface CategoryBadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof categoryBadgeVariants> {
  category: string;
}

/**
 * Shared component for displaying transaction or budget categories with consistent coloring.
 * Logic for determining category type is centralized here.
 */
export function CategoryBadge({ category, type, active, className, ...props }: CategoryBadgeProps) {
  // If type is not provided, we could optionally match it from a global list, 
  // but for now we expect it or default to expense.
  const resolvedType = type || (category === 'Uncategorized' ? 'uncategorized' : 'expense');

  return (
    <div className={cn(categoryBadgeVariants({ type: resolvedType, active }), className)} {...props}>
      {category}
    </div>
  );
}
