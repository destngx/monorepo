"use client";

import { MaskedBalance } from "@wealth-management/ui";
import { Calendar } from "lucide-react";

interface DueDateCountdownProps {
  balance: number;
  dueDay: number; // e.g. end of month = 31
}

export function DueDateCountdown({ balance, dueDay }: DueDateCountdownProps) {
  const now = new Date();
  const today = now.getDate();
  const lastDayOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
  
  const effectiveDueDay = dueDay > lastDayOfMonth ? lastDayOfMonth : dueDay;
  const minPayment = balance * 0.05; // Estimate 5% minimum
  
  let daysLeft: number;
  if (today <= effectiveDueDay) {
    daysLeft = effectiveDueDay - today;
  } else {
    // Due next month
    daysLeft = (lastDayOfMonth - today) + effectiveDueDay;
  }

  const isUrgent = daysLeft <= 5;

  return (
    <div className={`p-6 rounded-2xl border transition-all duration-500 ${
      isUrgent 
        ? "bg-rose-500/10 border-rose-500/50 shadow-[0_0_20px_rgba(244,63,94,0.1)]" 
        : "bg-emerald-500/5 border-emerald-500/20"
    }`}>
      <div className="flex items-center gap-2 mb-6">
        <div className={`p-2 rounded-lg ${isUrgent ? "bg-rose-500 text-white" : "bg-emerald-500 text-white"}`}>
          <Calendar className="h-5 w-5" />
        </div>
        <div>
          <h3 className="text-lg font-bold tracking-tight">Upcoming Due Date</h3>
          <p className={`text-xs font-bold uppercase tracking-widest ${isUrgent ? "text-rose-500" : "text-emerald-500"}`}>
            Payment due in {daysLeft} days
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1">
          <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Minimum Due</p>
          <p className="text-xl font-black tracking-tighter">
            <MaskedBalance amount={minPayment} />
          </p>
        </div>
        <div className="space-y-1">
          <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Full Balance</p>
          <p className="text-xl font-black tracking-tighter text-foreground">
            <MaskedBalance amount={balance} />
          </p>
        </div>
      </div>

      <div className="mt-6 pt-6 border-t border-dashed border-muted-foreground/20">
        <button className={`w-full py-3 rounded-xl font-bold text-sm transition-all ${
          isUrgent 
            ? "bg-rose-500 hover:bg-rose-600 text-white shadow-lg shadow-rose-500/25" 
            : "bg-foreground text-background hover:opacity-90"
        }`}>
          Pay Now
        </button>
      </div>
    </div>
  );
}
