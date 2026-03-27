"use client";

import { Sparkles, ArrowRight, BookOpen } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@wealth-management/ui/card";
import { cn } from "@wealth-management/utils";

interface AIInsightCardProps {
  title: string;
  time: string;
  content: string;
  onViewBreakdown: () => void;
  onAskFollowUp: () => void;
  className?: string;
}

export function AIInsightCard({
  title,
  time,
  content,
  onViewBreakdown,
  onAskFollowUp,
  className,
}: AIInsightCardProps) {
  return (
    <Card className={cn("overflow-hidden border-primary/20 bg-primary/5 shadow-md", className)}>
      <CardHeader className="py-3 px-4 flex flex-row items-center justify-between border-b border-primary/10 bg-primary/10">
        <CardTitle className="text-sm font-bold flex items-center gap-2 text-primary">
          <Sparkles className="h-4 w-4" />
          {title}
        </CardTitle>
        <span className="text-[10px] text-muted-foreground uppercase tracking-widest font-medium">
          {time}
        </span>
      </CardHeader>
      <CardContent className="p-4 space-y-4">
        <p className="text-sm leading-relaxed text-foreground/90 font-medium">
          {content}
        </p>
        <div className="flex gap-2 pt-1">
          <Button 
            variant="outline" 
            size="sm" 
            className="h-8 text-[11px] gap-1.5 border-primary/20 hover:bg-primary/10"
            onClick={onViewBreakdown}
          >
            <BookOpen className="h-3.5 w-3.5" />
            View Breakdown
          </Button>
          <Button 
            size="sm" 
            className="h-8 text-[11px] gap-1.5 bg-primary text-primary-foreground hover:opacity-90"
            onClick={onAskFollowUp}
          >
            Ask Follow-up
            <ArrowRight className="h-3.5 w-3.5" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
