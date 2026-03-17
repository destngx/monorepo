"use client";

import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import * as React from "react";

interface TransactionFiltersProps {
  search: string;
  onSearch: (q: string) => void;
}

export function TransactionFilters({ search, onSearch }: TransactionFiltersProps): React.JSX.Element {
  return (
    <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center py-4">
      <div className="relative w-full sm:max-w-sm">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          type="search"
          value={search}
          placeholder="Search payee, account, category, memo…"
          className="pl-8"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onSearch(e.currentTarget.value)}
        />
      </div>
    </div>
  );
}
