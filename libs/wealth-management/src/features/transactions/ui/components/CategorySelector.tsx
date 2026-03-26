"use client";

import { useState } from "react";
import { Search, Sparkles } from "lucide-react";
import { Button } from "@wealth-management/ui";
import { CategoryBadge } from "@wealth-management/ui";
import { cn } from "@wealth-management/utils";

interface CategoryChip {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

interface CategorySelectorProps {
  currentCategory: string;
  budgetCategories: CategoryChip[];
  isSuggesting: boolean;
  payee: string;
  setValue: any;
  onSuggest: () => void;
}

export function CategorySelector({ 
  currentCategory, 
  budgetCategories, 
  isSuggesting, 
  payee,
  setValue, 
  onSuggest 
}: CategorySelectorProps) {
  const [categorySearch, setCategorySearch] = useState("");
  const [showCategoryList, setShowCategoryList] = useState(false);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium">Category</label>
        <Button 
          type="button" 
          variant="ghost" 
          size="sm" 
          className="h-7 px-2 text-xs text-indigo-600 bg-indigo-50 hover:bg-indigo-100 cursor-pointer" 
          disabled={isSuggesting || !payee || payee.length < 3}
          onClick={onSuggest}
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
    </div>
  );
}
