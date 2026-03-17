"use client";

import { Sparkles, Loader2 } from "lucide-react";
import { cn } from "@wealth-management/utils";
import { useScrollState } from "@/hooks/use-scroll-state";
import { useState, useEffect } from "react";

interface AIFabProps {
  onClick: () => void;
  isLoading?: boolean;
  hasInsight?: boolean;
  isOpen?: boolean;
}

export function AIFab({ onClick, isLoading, hasInsight, isOpen }: AIFabProps) {
  const { isScrollingDown } = useScrollState();
  const [isMinimized, setIsMinimized] = useState(false);

  // Re-expand on scroll stop or scroll up
  useEffect(() => {
    if (isScrollingDown) {
      setIsMinimized(true);
    } else {
      setIsMinimized(false);
    }
  }, [isScrollingDown]);

  if (isOpen) return null;

  return (
    <button
      onClick={onClick}
      className={cn(
        "fixed bottom-6 right-6 z-50 flex h-12 items-center justify-center rounded-full shadow-lg transition-all duration-500 ease-in-out cursor-pointer overflow-hidden",
        "bg-primary text-primary-foreground hover:scale-105 active:scale-95",
        isMinimized ? "w-12" : "w-[124px] px-4",
        hasInsight && !isLoading && "ring-4 ring-primary/20 animate-pulse"
      )}
      aria-label="Ask AI"
    >
      <div className="flex items-center gap-2 whitespace-nowrap">
        {isLoading ? (
          <Loader2 className="h-5 w-5 animate-spin" />
        ) : (
          <div className="relative">
            <Sparkles className={cn("h-5 w-5", hasInsight && "text-amber-300")} />
            {hasInsight && (
              <span className="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-amber-500"></span>
              </span>
            )}
          </div>
        )}
        <span 
          className={cn(
            "font-semibold text-sm transition-all duration-300",
            isMinimized ? "opacity-0 w-0 scale-0" : "opacity-100 w-auto scale-100"
          )}
        >
          Ask AI
        </span>
      </div>
      
      {/* Background glow for insight state */}
      {hasInsight && (
        <div className="absolute inset-0 -z-10 bg-primary blur-xl opacity-20 animate-pulse" />
      )}
    </button>
  );
}
