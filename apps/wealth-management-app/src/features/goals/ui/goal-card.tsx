"use client";

import { Card, CardContent } from "@/components/ui/card";
import { Goal } from '../model/types';
import { Progress } from "@/components/ui/progress";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import { cn } from '@wealth-management/utils';

interface GoalCardProps {
  goal: Goal;
}

export function GoalCard({ goal }: GoalCardProps) {
  const progress = Math.min((goal.currentAmount / goal.targetAmount) * 100, 100);
  
  const statusColors = {
    ON_TRACK: "bg-emerald-500/10 text-emerald-500 border-emerald-500/20",
    AT_RISK: "bg-amber-500/10 text-amber-500 border-amber-500/20",
    OFF_TRACK: "bg-rose-500/10 text-rose-500 border-rose-500/20",
  };

  const statusLabels = {
    ON_TRACK: "On Track",
    AT_RISK: "At Risk",
    OFF_TRACK: "Off Track",
  };

  return (
    <Link href={`/accounts/goals/${goal.id}`}>
      <Card className="group hover:border-indigo-500/50 transition-all duration-300 hover:shadow-lg hover:shadow-indigo-500/5 cursor-pointer">
        <CardContent className="p-6">
          <div className="flex justify-between items-start mb-6">
            <div className="flex items-center gap-3">
              <div className="text-3xl grayscale group-hover:grayscale-0 transition-all duration-300">
                {goal.emoji}
              </div>
              <div>
                <h3 className="font-semibold group-hover:text-indigo-500 transition-colors uppercase tracking-tight text-sm">
                  {goal.name}
                </h3>
                <p className="text-xs text-muted-foreground">
                  {goal.streakCount > 0 && `${goal.streakCount}-week streak 🔥`}
                </p>
              </div>
            </div>
            <Badge variant="outline" className={cn("text-[10px] font-bold uppercase py-0 px-1.5", statusColors[goal.status])}>
              {statusLabels[goal.status]}
            </Badge>
          </div>

          <div className="flex flex-col items-center justify-center mb-6 relative">
             {/* Simple Ring abstraction for now, using progress for foundation */}
             <div className="relative w-32 h-32 flex items-center justify-center">
                <svg className="w-full h-full -rotate-90">
                  <circle
                    cx="64"
                    cy="64"
                    r="58"
                    stroke="currentColor"
                    strokeWidth="8"
                    fill="transparent"
                    className="text-muted/10"
                  />
                  <circle
                    cx="64"
                    cy="64"
                    r="58"
                    stroke="currentColor"
                    strokeWidth="8"
                    fill="transparent"
                    strokeDasharray={364.4}
                    strokeDashoffset={364.4 - (364.4 * progress) / 100}
                    strokeLinecap="round"
                    className="text-indigo-500 transition-all duration-1000 ease-in-out"
                  />
                </svg>
                <div className="absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-2xl font-bold">{progress.toFixed(0)}%</span>
                </div>
             </div>
          </div>

          <div className="space-y-4">
            <div className="flex justify-between text-xs">
              <span className="text-muted-foreground font-medium">
                <MaskedBalance amount={goal.currentAmount} />
              </span>
              <span className="text-muted-foreground">
                <MaskedBalance amount={goal.targetAmount} />
              </span>
            </div>
            
            <div className="flex justify-between items-center bg-muted/30 p-2 rounded-lg">
              <span className="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Deadline</span>
              <span className="text-xs font-semibold">{goal.deadline}</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
