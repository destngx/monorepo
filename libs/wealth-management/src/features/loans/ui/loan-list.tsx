"use client";

import { Loan } from "../model/types";
import { MaskedBalance } from "@wealth-management/ui";
import { Progress } from "@wealth-management/ui";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@wealth-management/ui";

export function LoanList({ loans }: { loans: Loan[] }) {
  return (
    <div className="rounded-xl border bg-card shadow-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-muted/50">
            <TableHead className="font-semibold">Lender/Debt</TableHead>
            <TableHead className="text-right font-semibold">Month Paid</TableHead>
            <TableHead className="text-right font-semibold">Month Total</TableHead>
            <TableHead className="font-semibold hidden lg:table-cell">Monthly Progress</TableHead>
            <TableHead className="text-right font-semibold">Yearly Total</TableHead>
            <TableHead className="text-right font-semibold">Yearly Paid</TableHead>
            <TableHead className="text-right font-semibold">Yearly Rem.</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loans.map((loan) => {
            const progress = loan.monthlyDebt > 0 ? (loan.monthlyPaid / loan.monthlyDebt) * 100 : 0;
            
            return (
              <TableRow key={loan.name} className="hover:bg-muted/30 transition-colors">
                <TableCell className="font-medium whitespace-nowrap">
                  <MaskedBalance value={loan.name} currency="none" />
                </TableCell>
                <TableCell className="text-right text-green-600 dark:text-green-400">
                  <MaskedBalance amount={loan.monthlyPaid} />
                </TableCell>
                <TableCell className="text-right font-semibold">
                  <MaskedBalance amount={loan.monthlyDebt} />
                </TableCell>
                <TableCell className="hidden lg:table-cell w-40">
                  <div className="flex flex-col gap-1">
                    <Progress value={progress} className="h-1.5" />
                    <span className="text-[10px] text-muted-foreground text-right">{Math.round(progress)}%</span>
                  </div>
                </TableCell>
                <TableCell className="text-right text-muted-foreground">
                  <MaskedBalance amount={loan.yearlyDebt} />
                </TableCell>
                <TableCell className="text-right text-green-600/80">
                  <MaskedBalance amount={loan.yearlyPaid} />
                </TableCell>
                <TableCell className="text-right font-bold text-orange-600 dark:text-orange-400">
                  <MaskedBalance amount={loan.yearlyRemaining} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
