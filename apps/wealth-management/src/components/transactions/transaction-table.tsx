"use client";

import { useState } from "react";
import { Transaction } from "@wealth-management/types";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { formatDate } from "@wealth-management/utils";
import { CategoryBadge } from "@/components/ui/category-badge";
import { ArrowUp, ArrowDown, ArrowUpDown } from "lucide-react";

type SortKey = "date" | "accountName" | "payee" | "category" | "payment" | "deposit";
type SortDir = "asc" | "desc";

function SortIcon({ col, sortKey, dir }: { col: SortKey; sortKey: SortKey; dir: SortDir }) {
  if (col !== sortKey) return <ArrowUpDown className="h-3 w-3 opacity-40 ml-1 inline" />;
  return dir === "asc"
    ? <ArrowUp className="h-3 w-3 ml-1 inline text-primary" />
    : <ArrowDown className="h-3 w-3 ml-1 inline text-primary" />;
}

function sortTransactions(txns: Transaction[], key: SortKey, dir: SortDir): Transaction[] {
  return [...txns].sort((a, b) => {
    let av: string | number | Date = "";
    let bv: string | number | Date = "";
    switch (key) {
      case "date":       av = new Date(a.date).getTime(); bv = new Date(b.date).getTime(); break;
      case "accountName":av = a.accountName.toLowerCase(); bv = b.accountName.toLowerCase(); break;
      case "payee":      av = (a.payee || "").toLowerCase(); bv = (b.payee || "").toLowerCase(); break;
      case "category":   av = (a.category || "").toLowerCase(); bv = (b.category || "").toLowerCase(); break;
      case "payment":    av = a.payment ?? 0; bv = b.payment ?? 0; break;
      case "deposit":    av = a.deposit ?? 0; bv = b.deposit ?? 0; break;
    }
    if (av < bv) return dir === "asc" ? -1 : 1;
    if (av > bv) return dir === "asc" ? 1 : -1;
    return 0;
  });
}

interface TransactionTableProps {
  transactions: Transaction[];
  onSortChange?: (key: SortKey, dir: SortDir) => void;
}

export function TransactionTable({ transactions }: TransactionTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>("date");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const handleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir(d => d === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const sorted = sortTransactions(transactions, sortKey, sortDir);

  if (transactions.length === 0) {
    return <div className="p-8 text-center text-muted-foreground">No transactions found.</div>;
  }

  const thClass = (col: SortKey, right = false) =>
    `px-4 py-3 font-medium cursor-pointer select-none whitespace-nowrap hover:text-foreground transition-colors ${right ? "text-right" : ""}`;

  return (
    <div className="w-full relative overflow-x-auto border rounded-md">
      <table className="w-full text-sm text-left">
        <thead className="bg-muted text-muted-foreground uppercase text-xs">
          <tr>
            <th className={thClass("date")} onClick={() => handleSort("date")}>
              Date <SortIcon col="date" sortKey={sortKey} dir={sortDir} />
            </th>
            <th className={thClass("accountName")} onClick={() => handleSort("accountName")}>
              Account <SortIcon col="accountName" sortKey={sortKey} dir={sortDir} />
            </th>
            <th className={thClass("payee")} onClick={() => handleSort("payee")}>
              Payee / Memo <SortIcon col="payee" sortKey={sortKey} dir={sortDir} />
            </th>
            <th className={thClass("category")} onClick={() => handleSort("category")}>
              Category <SortIcon col="category" sortKey={sortKey} dir={sortDir} />
            </th>
            <th className={thClass("payment", true)} onClick={() => handleSort("payment")}>
              Out (Payment) <SortIcon col="payment" sortKey={sortKey} dir={sortDir} />
            </th>
            <th className={thClass("deposit", true)} onClick={() => handleSort("deposit")}>
              In (Deposit) <SortIcon col="deposit" sortKey={sortKey} dir={sortDir} />
            </th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((txn, i) => (
            <tr key={txn.id || i} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
              <td className="px-4 py-3 whitespace-nowrap">{formatDate(new Date(txn.date))}</td>
              <td className="px-4 py-3 whitespace-nowrap">
                <div>{txn.accountName}</div>
                {txn.tags && txn.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-1">
                    {txn.tags.map(tag => {
                      const lowerTag = tag.toLowerCase();
                      const isUniq = lowerTag.includes('uniq');
                      const isPlatinum = lowerTag.includes('platinum');
                      
                      const tagClass = isUniq 
                        ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400" 
                        : isPlatinum 
                        ? "bg-blue-500/10 text-blue-600 dark:text-blue-400" 
                        : "bg-primary/10 text-primary";

                      return (
                        <span key={tag} className={`inline-block ${tagClass} text-[10px] font-bold px-1.5 py-0.5 rounded-md border border-current/10 uppercase tracking-tighter`}>
                          {tag}
                        </span>
                      );
                    })}
                  </div>
                )}
              </td>
              <td className="px-4 py-3 min-w-[200px]">
                <div className="font-medium text-foreground">{txn.payee}</div>
                {txn.memo && <div className="text-xs text-muted-foreground">{txn.memo}</div>}
              </td>
              <td className="px-4 py-3">
                <CategoryBadge category={txn.category} type={txn.categoryType} />
              </td>
              <td className="px-4 py-3 text-right text-orange-500 font-medium tabular-nums whitespace-nowrap">
                {txn.payment ? <><span className="mr-0.5">-</span><MaskedBalance amount={txn.payment} /></> : '-'}
              </td>
              <td className="px-4 py-3 text-right text-emerald-600 font-medium tabular-nums whitespace-nowrap">
                {txn.deposit ? <><span className="mr-0.5">+</span><MaskedBalance amount={txn.deposit} /></> : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
