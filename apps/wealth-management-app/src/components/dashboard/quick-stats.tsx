import { Transaction } from "@wealth-management/types";

import { MaskedBalance } from "@/components/ui/masked-balance";
import { getEffectiveDate } from "@wealth-management/utils";

const MONTH_NAMES = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

function getMonthTxns(transactions: Transaction[], month: number, year: number) {
  return transactions.filter(t => {
    const txDate = getEffectiveDate(t);
    return txDate.getMonth() === month && txDate.getFullYear() === year;
  });
}

export function QuickStats({ transactions }: { transactions: Transaction[] }) {
  const now = new Date();
  let month = now.getMonth();
  let year = now.getFullYear();

  let monthTxns = getMonthTxns(transactions, month, year);
  let label = 'This Month';

  if (monthTxns.length === 0) {
    month = month === 0 ? 11 : month - 1;
    year = month === 11 ? year - 1 : year;
    monthTxns = getMonthTxns(transactions, month, year);
    label = MONTH_NAMES[month];
  }

  const income = monthTxns.reduce((sum, t) => {
    if (t.categoryType === 'income') return sum + (t.deposit || 0);
    return sum;
  }, 0);

  const expense = monthTxns.reduce((sum, t) => {
    if (t.categoryType === 'expense') return sum + (t.payment || 0);
    return sum;
  }, 0);
  const net = income - expense;
  const savingsRate = income > 0 ? ((income - expense) / income) * 100 : 0;

  return (
    <div className="flex flex-wrap items-center gap-0 border rounded-lg bg-card overflow-hidden divide-x w-full">
      <div className="flex-1 min-w-[120px] px-4 py-2 flex items-baseline justify-between gap-2">
        <span className="text-[10px] font-medium text-muted-foreground uppercase">Inc ({label})</span>
        <span className="text-sm font-bold text-emerald-600 whitespace-nowrap"><MaskedBalance amount={income} /></span>
      </div>
      <div className="flex-1 min-w-[120px] px-4 py-2 flex items-baseline justify-between gap-2">
        <span className="text-[10px] font-medium text-muted-foreground uppercase">Exp</span>
        <span className="text-sm font-bold text-orange-600 whitespace-nowrap"><MaskedBalance amount={expense} /></span>
      </div>
      <div className="flex-1 min-w-[120px] px-4 py-2 flex items-baseline justify-between gap-2">
        <span className="text-[10px] font-medium text-muted-foreground uppercase">Net</span>
        <span className={`text-sm font-bold whitespace-nowrap ${net < 0 ? 'text-orange-500' : 'text-primary'}`}><MaskedBalance amount={net} /></span>
      </div>
      <div className="flex-1 min-w-[120px] px-4 py-2 flex items-baseline justify-between gap-2">
        <span className="text-[10px] font-medium text-muted-foreground uppercase">Savings</span>
        <span className="text-sm font-bold whitespace-nowrap">{savingsRate.toFixed(1)}%</span>
      </div>
    </div>
  );
}
