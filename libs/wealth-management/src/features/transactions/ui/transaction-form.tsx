"use client";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@wealth-management/ui";
import { Button } from "@wealth-management/ui";
import { Input } from "@wealth-management/ui";
import { TransactionInput } from "@wealth-management/schemas";
import { useTransactionForm } from "./hooks/useTransactionForm";
import { CategorySelector } from "./components/CategorySelector";
import { TagSelector } from "./components/TagSelector";

export function TransactionForm({ 
  open, 
  onOpenChange,
  onSubmit
}: { 
  open: boolean; 
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: TransactionInput) => Promise<void>;
}) {
  const {
    form,
    loading,
    accounts,
    availableTags,
    budgetCategories,
    isSuggesting,
    onFormSubmit,
    handleClose,
    handleSuggestCategory,
    addTag,
    removeTag
  } = useTransactionForm(open, onOpenChange, onSubmit);

  const { register, handleSubmit, formState: { errors }, watch, setValue } = form;
  const payee = watch("payee");
  const currentCategory = watch("category");
  const selectedTags = watch("tags") ?? [];

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); }}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Add Transaction</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4 pt-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <label className="text-sm font-medium">Account</label>
              <select
                {...register("accountName")}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="">Select account...</option>
                {accounts.map((acc, i) => (
                  <option key={i} value={acc.name}>{acc.name}</option>
                ))}
              </select>
              {errors.accountName && <span className="text-xs text-red-500">{errors.accountName.message}</span>}
            </div>
            <div className="space-y-1">
              <label className="text-sm font-medium">Date</label>
              <Input type="date" {...register("date", { valueAsDate: true })} />
              {errors.date && <span className="text-xs text-red-500">{errors.date.message}</span>}
            </div>
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Payee</label>
            <Input {...register("payee")} placeholder="Merchant name" />
            {errors.payee && <span className="text-xs text-red-500">{errors.payee.message}</span>}
          </div>

          <CategorySelector 
            currentCategory={currentCategory}
            budgetCategories={budgetCategories}
            isSuggesting={isSuggesting}
            payee={payee}
            setValue={setValue}
            onSuggest={() => handleSuggestCategory(payee)}
          />
          {errors.category && <span className="text-xs text-red-500">{errors.category.message}</span>}

          <TagSelector 
            selectedTags={selectedTags}
            availableTags={availableTags}
            addTag={addTag}
            removeTag={removeTag}
          />

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <label className="text-sm font-medium">Payment (Out)</label>
              <Input type="number" step="0.01" {...register("payment", { valueAsNumber: true })} />
            </div>
            <div className="space-y-1">
              <label className="text-sm font-medium">Deposit (In)</label>
              <Input type="number" step="0.01" {...register("deposit", { valueAsNumber: true })} />
            </div>
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Memo</label>
            <Input {...register("memo")} placeholder="Optional note" />
          </div>

          <div className="pt-4 flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? "Thinking..." : "Save Transaction"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
