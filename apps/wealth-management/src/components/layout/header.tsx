'use client';

import { usePathname } from 'next/navigation';
import { RefreshCw, Menu, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ThemeToggle } from '@/components/theme-toggle';
import { useState, useEffect, useCallback, useRef } from 'react';
import { useAISettings } from '@/hooks/use-ai-settings';
import { Sheet, SheetContent, SheetTrigger, SheetTitle, SheetHeader } from '@/components/ui/sheet';
import { NAV_LINKS } from '@wealth-management/utils';
import Link from 'next/link';
import { cn } from '@wealth-management/utils';
import { ModelSwitcher } from '@/features/chat/ui/model-switcher';
import { useMask } from '@wealth-management/ui/mask-provider';

const SYNC_INTERVAL_MS = 15 * 60 * 1000; // 15 minutes

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/transactions': 'Transactions',
  '/budget': 'Budget',
  '/accounts': 'Accounts',
  '/investments': 'Investments',
  '/credit-cards': 'Credit Card',
  '/goals': 'Financial Goals',
  '/chat': 'AI Wealth Advisor',
  '/settings': 'Settings',
};

function formatTime(date: Date) {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatCountdown(ms: number) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export function Header() {
  const { isMasked, toggleMask } = useMask();
  const pathname = usePathname();
  const [syncing, setSyncing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState<Date | null>(null);
  const [nextSyncIn, setNextSyncIn] = useState(SYNC_INTERVAL_MS);
  const nextSyncAtRef = useRef<number>(Date.now() + SYNC_INTERVAL_MS);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const title =
    pageTitles[pathname] ||
    Object.keys(pageTitles)
      .find((key) => key !== '/' && pathname.startsWith(key))
      ?.replace('/', '') ||
    'WealthOS';

  const doSync = useCallback(async () => {
    if (syncing) return;
    setSyncing(true);
    try {
      await fetch('/api/sync', { method: 'POST' });
      setLastSyncedAt(new Date());
    } catch {
      // silent — next auto-sync will retry
    } finally {
      setSyncing(false);
      // Reset countdown
      nextSyncAtRef.current = Date.now() + SYNC_INTERVAL_MS;
      setNextSyncIn(SYNC_INTERVAL_MS);
    }
  }, [syncing]);

  // Auto-sync on mount + every 15 minutes
  useEffect(() => {
    void doSync(); // initial sync when app loads

    const autoInterval = setInterval(() => {
      void doSync();
    }, SYNC_INTERVAL_MS);

    return () => clearInterval(autoInterval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Countdown ticker — updates every second
  useEffect(() => {
    const ticker = setInterval(() => {
      const remaining = nextSyncAtRef.current - Date.now();
      setNextSyncIn(Math.max(0, remaining));
    }, 1000);
    return () => clearInterval(ticker);
  }, []);

  const { mounted } = useAISettings();

  return (
    <header className="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b bg-background/95 px-4 md:px-6 backdrop-blur">
      <div className="flex items-center gap-3">
        <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon" className="md:hidden cursor-pointer">
              <Menu className="h-5 w-5" />
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="w-64 p-0">
            <SheetHeader>
              <SheetTitle>
                <div className="flex h-12 items-center px-2">
                  <h1 className="text-lg font-bold">WealthOS</h1>
                </div>
              </SheetTitle>
            </SheetHeader>
            <nav className="flex flex-col gap-2 p-4">
              {NAV_LINKS.map((link) => {
                const isActive = pathname === link.href || (link.href !== '/' && pathname.startsWith(link.href));
                return (
                  <Link
                    key={link.href}
                    href={link.href}
                    onClick={() => setMobileMenuOpen(false)}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-muted hover:text-foreground',
                      isActive ? 'bg-muted text-foreground' : 'text-muted-foreground',
                    )}
                  >
                    <link.icon className="h-4 w-4" />
                    {link.label}
                  </Link>
                );
              })}
            </nav>
          </SheetContent>
        </Sheet>
        <h2 className="text-base md:text-lg font-semibold truncate max-w-[150px] sm:max-w-[300px]">{title}</h2>
      </div>

      <div className="flex items-center gap-4">
        {mounted && (
          <div className="hidden md:flex items-center mr-2">
            <ModelSwitcher />
          </div>
        )}

        <div className="flex flex-col items-end gap-0.5 hidden sm:flex">
          <div className="flex items-center gap-2 text-[11px] sm:text-sm text-muted-foreground">
            <span className="relative flex h-2 w-2">
              {syncing ? (
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75" />
              ) : (
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
              )}
              <span
                className={`relative inline-flex h-2 w-2 rounded-full ${syncing ? 'bg-amber-500' : 'bg-green-500'}`}
              />
            </span>
            <span className="whitespace-nowrap">
              {syncing ? 'Syncing…' : lastSyncedAt ? `Synced: ${formatTime(lastSyncedAt)}` : 'Not synced'}
            </span>
          </div>
          {!syncing && (
            <span className="text-[10px] text-muted-foreground/60 leading-none pr-0.5 whitespace-nowrap">
              Next in {formatCountdown(nextSyncIn)}
            </span>
          )}
        </div>

        <ThemeToggle />

        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 px-0 cursor-pointer"
          onClick={toggleMask}
          title={isMasked ? 'Show all values' : 'Hide all values'}
        >
          {isMasked ? <EyeOff className="h-4 w-4 text-muted-foreground" /> : <Eye className="h-4 w-4 text-primary" />}
        </Button>

        <Button
          variant="outline"
          size="sm"
          className="gap-2 cursor-pointer h-8 px-2 sm:px-3 sm:h-9"
          onClick={doSync}
          disabled={syncing}
        >
          <RefreshCw className={`h-4 w-4 ${syncing ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">Sync</span>
        </Button>
      </div>
    </header>
  );
}
