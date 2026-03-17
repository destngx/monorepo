"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { 
  ChevronLeft,
  ChevronRight
} from "lucide-react";
import { cn } from "@wealth-management/utils";
import { NAV_LINKS } from "@wealth-management/utils";
import { useSidebar } from "./sidebar-provider";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function Sidebar() {
  const pathname = usePathname();
  const { isCollapsed, toggleSidebar } = useSidebar();

  return (
    <aside 
      className={cn(
        "hidden md:flex flex-col fixed left-0 top-0 z-40 h-screen overflow-hidden transition-all duration-300 ease-in-out border-r bg-background",
        isCollapsed ? "w-16" : "w-64"
      )}
    >
      <div className={cn(
        "flex h-16 items-center border-b transition-all duration-300",
        isCollapsed ? "justify-center px-0" : "px-6"
      )}>
        {isCollapsed ? (
          <span className="text-xl font-bold">W</span>
        ) : (
          <h1 className="text-lg font-bold truncate">WealthOS</h1>
        )}
      </div>

      <nav className="flex flex-col gap-2 p-3 flex-1 overflow-y-auto overflow-x-hidden">
        <TooltipProvider delayDuration={0}>
          {NAV_LINKS.map((link) => {
            const isActive = pathname === link.href || (link.href !== "/" && pathname.startsWith(link.href));
            
            const linkContent = (
              <Link
                href={link.href}
                className={cn(
                  "flex items-center rounded-md transition-colors hover:bg-muted hover:text-foreground h-10",
                  isCollapsed ? "justify-center px-0 w-10 mx-auto" : "gap-3 px-3 w-full",
                  isActive ? "bg-muted text-foreground" : "text-muted-foreground"
                )}
              >
                <link.icon className="h-4 w-4 shrink-0" />
                {!isCollapsed && <span className="truncate">{link.label}</span>}
              </Link>
            );

            if (isCollapsed) {
              return (
                <Tooltip key={link.href}>
                  <TooltipTrigger asChild>
                    {linkContent}
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    <p>{link.label}</p>
                  </TooltipContent>
                </Tooltip>
              );
            }

            return <div key={link.href}>{linkContent}</div>;
          })}
        </TooltipProvider>
      </nav>

      <div className="p-3 border-t">
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleSidebar}
          className="w-full justify-center h-10 cursor-pointer"
          title={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {isCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
          {!isCollapsed && <span className="ml-2 text-sm">Collapse</span>}
        </Button>
      </div>
    </aside>
  );
}
