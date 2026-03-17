"use client";

import { useSidebar } from "./sidebar-provider";
import { cn } from "@wealth-management/utils";

export function LayoutWrapper({ children }: { children: React.ReactNode }) {
  const { isCollapsed } = useSidebar();

  return (
    <div className={cn(
      "flex w-full flex-col transition-all duration-300 ease-in-out",
      isCollapsed ? "md:pl-16" : "md:pl-64"
    )}>
      {children}
    </div>
  );
}
