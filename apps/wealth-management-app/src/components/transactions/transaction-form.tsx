"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { TransactionSchema, TransactionInput } from "@wealth-management/schemas";
import { Account } from "@wealth-management/types";
import { X, Sparkles, Search } from "lucide-react";
import { cn } from "@wealth-management/utils";
import { CategoryBadge } from "@/components/ui/category-badge";

interface CategoryChip {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

export function TransactionForm({ 
  open, 
  onOpenChange,
  onSubmit
}: { 
  open: boolean; 
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: TransactionInput) => Promise<void>;
}) {
  const [loading, setLoading] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [budgetCategories, setBudgetCategories] = useState<CategoryChip[]>([]);
  const [categorySearch, setCategorySearch] = useState("");
  const [isSuggesting, setIsSuggesting] = useState(false);
  const [showCategoryList, setShowCategoryList] = useState(false);

  useEffect(() => {
    if (open) {
      fetch('/api/accounts')
        .then(r => r.json())
        .then(data => {
          setAccounts(data);
          // Always try to set Golden Pocket if it's in the list
          if (data.some((a: Account) => a.name === "Golden Pocket")) {
            setValue("accountName", "Golden Pocket");
          }
        })
        .catch(console.error);

      fetch('/api/tags')
        .then(r => r.json())
        .then(data => Array.isArray(data) ? setAvailableTags(data) : null)
        .catch(console.error);

      fetch('/api/categories')
        .then(r => r.json())
        .then((data: CategoryChip[]) => {
          if (Array.isArray(data) && data.length > 0) {
            setBudgetCategories(data);
          }
        })
        .catch(console.error);
    }
  }, [open]);

  const { register, handleSubmit, formState: { errors }, reset, watch, setValue } = useForm<TransactionInput>({
    resolver: zodResolver(TransactionSchema as any),
    defaultValues: {
      accountName: "Golden Pocket",
      date: new Date().toISOString().split('T')[0] as any,
      payee: "",
      category: "",
      payment: null,
      deposit: null,
      cleared: false,
      tags: [],
    }
  });

  const payee = watch("payee");
  const currentCategory = watch("category");

  // AI Suggestion Logic
  const handleSuggestCategory = async (targetPayee: string) => {
    if (!targetPayee || targetPayee.length < 3 || budgetCategories.length === 0) return;
    
    setIsSuggesting(true);
    try {
      const res = await fetch('/api/ai/suggest-category', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payee: targetPayee, categories: budgetCategories.map(c => c.name) })
      });
      const data = await res.json();
      if (data.category) {
        setValue("category", data.category);
      }
    } catch (error) {
      console.error("AI Suggestion Error:", error);
    } finally {
      setIsSuggesting(false);
    }
  };

  // Debounced Auto-suggestion
  useEffect(() => {
    if (!payee || payee.length < 3 || currentCategory) return;

    const timer = setTimeout(() => {
      handleSuggestCategory(payee);
    }, 1000); // 1s debounce

    return () => clearTimeout(timer);
  }, [payee, budgetCategories]);

  const selectedTags = watch("tags") ?? [];

  const addTag = (value: string) => {
    if (!value || selectedTags.includes(value)) return;
    setValue("tags", [...selectedTags, value]);
  };

  const removeTag = (tag: string) => {
    setValue("tags", selectedTags.filter((t: string) => t !== tag));
  };

  const handleClose = () => {
    reset();
    onOpenChange(false);
  };

  const onFormSubmit = async (data: TransactionInput) => {
    setLoading(true);
    try {
      await onSubmit(data);
      reset();
      onOpenChange(false);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

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

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Category</label>
              <Button 
                type="button" 
                variant="ghost" 
                size="sm" 
                className="h-7 px-2 text-xs text-indigo-600 bg-indigo-50 hover:bg-indigo-100 cursor-pointer" 
                disabled={isSuggesting || !payee || payee.length < 3}
                onClick={() => handleSuggestCategory(payee)}
              >
                {isSuggesting ? (
                  <Sparkles className="h-3 w-3 animate-pulse mr-1" />
                ) : (
                  <Sparkles className="h-3 w-3 mr-1" />
                )}
                Suggest
              </Button>
            </div>
            
            <div className="relative space-y-2">
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  type="text"
                  placeholder="Search categories..."
                  value={categorySearch}
                  onFocus={() => setShowCategoryList(true)}
                  onBlur={() => setTimeout(() => setShowCategoryList(false), 200)}
                  onChange={(e) => {
                    setCategorySearch(e.target.value);
                    setShowCategoryList(true);
                  }}
                  className="h-9 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                />
              </div>

              {currentCategory && (
                <div className="flex items-center gap-2 px-1">
                  <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Active:</span>
                  <CategoryBadge 
                    category={currentCategory} 
                    type={budgetCategories.find(c => c.name === currentCategory)?.type}
                    active 
                    className="px-3 py-1"
                  />
                  <button 
                    type="button" 
                    onClick={() => setValue("category", "")}
                    className="text-[10px] font-bold text-red-500 hover:text-red-600 uppercase tracking-wider transition-colors"
                  >
                    Clear
                  </button>
                </div>
              )}

              {showCategoryList && (
                <div className="absolute top-full left-0 z-50 mt-1 w-full border rounded-md max-h-[250px] overflow-y-auto bg-popover shadow-lg divide-y divide-border animate-in fade-in zoom-in-95 duration-100">
                  {budgetCategories
                    .filter(cat => !categorySearch || cat.name.toLowerCase().includes(categorySearch.toLowerCase()))
                    .map(cat => {
                      const isSelected = currentCategory === cat.name;
                      return (
                        <button
                          key={cat.name}
                          type="button"
                          onClick={() => {
                            setValue("category", cat.name);
                            setCategorySearch("");
                            setShowCategoryList(false);
                          }}
                          className={cn(
                            "w-full text-left px-3 py-2.5 transition-colors hover:bg-accent flex items-center justify-between group",
                            isSelected && "bg-accent/50"
                          )}
                        >
                          <CategoryBadge 
                            category={cat.name} 
                            type={cat.type} 
                            active={false}
                            className="border-0 bg-transparent p-0 text-sm"
                          />
                          {isSelected ? (
                            <span className="text-[10px] font-bold text-indigo-600 uppercase tracking-tighter">Selected</span>
                          ) : (
                            <span className="text-[10px] font-bold text-muted-foreground/0 group-hover:text-muted-foreground/50 uppercase tracking-tighter transition-opacity">Select</span>
                          )}
                        </button>
                      );
                    })}
                  {budgetCategories.length === 0 && (
                    <p className="text-[11px] text-muted-foreground w-full text-center py-4">Thinking...</p>
                  )}
                  {budgetCategories.length > 0 && budgetCategories.filter(cat => !categorySearch || cat.name.toLowerCase().includes(categorySearch.toLowerCase())).length === 0 && (
                    <p className="text-[11px] text-muted-foreground w-full text-center py-4">No categories match your search</p>
                  )}
                </div>
              )}
            </div>
            {errors.category && <span className="text-xs text-red-500">{errors.category.message}</span>}
          </div>

          {/* Tags */}
          <div className="space-y-1">
            <label className="text-sm font-medium">Tags</label>

            {/* Selected tag chips */}
            {selectedTags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-1.5">
                {selectedTags.map(tag => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 bg-primary/10 text-primary text-xs font-medium px-2 py-0.5 rounded-full"
                  >
                    {tag}
                    <button
                      type="button"
                      onClick={() => removeTag(tag)}
                      className="hover:text-destructive transition-colors cursor-pointer"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}

            {/* Dynamic select from existing tags in the sheet */}
            <select
              value=""
              onChange={e => { addTag(e.target.value); (e.target as HTMLSelectElement).value = ""; }}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              disabled={availableTags.length === 0}
            >
              <option value="">
                {availableTags.length === 0 ? "Thinking…" : "Add a tag…"}
              </option>
              {availableTags.filter(t => !selectedTags.includes(t)).map(tag => (
                <option key={tag} value={tag}>{tag}</option>
              ))}
            </select>
          </div>

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
