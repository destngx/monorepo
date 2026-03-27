"use client";

import { Progress } from "-management/ui/progress";
import { Clock } from "lucide-react";

interface StatementCycleBarProps {
  reportDay: number; // e.g., 5th of month
}

export function StatementCycleBar({ reportDay }: StatementCycleBarProps) {
  const now = new Date();
  const today = now.getDate();
  
  let daysInCycle: number;
  let elapsed: number;
  
  const lastMonth = new Date(now.getFullYear(), now.getMonth(), 0).getDate();
  const thisMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();

  if (today >= reportDay) {
    // Current cycle started this month
    elapsed = today - reportDay;
    daysInCycle = thisMonth; // Approximate
  } else {
    // Current cycle started last month
    elapsed = (lastMonth - reportDay) + today;
    daysInCycle = lastMonth; // Approximate
  }

  const progress = Math.min(100, (elapsed / daysInCycle) * 100);
  const remainingDays = Math.max(0, daysInCycle - elapsed);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Billing Cycle</span>
        </div>
        <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
          {remainingDays} days left
        </span>
      </div>
      
      <div className="space-y-2">
        <Progress value={progress} className="h-2" />
        <div className="flex justify-between text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
          <span>Day {elapsed}</span>
          <span>Day {daysInCycle}</span>
        </div>
      </div>

      <div className="p-3 rounded-lg bg-muted/30 border border-dashed text-center">
        <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest mb-1">Estimated Statement</p>
        <p className="text-lg font-bold tracking-tight">Calculating...</p>
        <p className="text-[9px] text-muted-foreground mt-1 italic">Based on current pending transactions</p>
      </div>
    </div>
  );
}
