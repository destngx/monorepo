import { Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";

interface LoadingProps extends React.HTMLAttributes<HTMLDivElement> {
  message?: string;
  fullScreen?: boolean;
}

export function Loading({ message = "Thinking...", fullScreen = false, className, ...props }: LoadingProps) {
  return (
    <div 
      className={cn(
        "flex flex-col items-center justify-center gap-3 text-muted-foreground",
        fullScreen ? "h-[50vh] min-h-[400px]" : "py-12",
        className
      )}
      {...props}
    >
      <Sparkles className="h-8 w-8 animate-pulse text-indigo-500" />
      <p className="text-sm font-medium animate-pulse">{message}</p>
    </div>
  );
}
