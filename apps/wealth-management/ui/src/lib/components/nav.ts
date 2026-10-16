import type { IconName } from './Icon.svelte';

export interface NavItem {
  href: string;
  label: string;
  icon: IconName;
  children?: NavItem[];
  /** A count shown next to the label, e.g. drafts waiting for review. */
  badge?: 'pendingDrafts';
}

// Navigation order from WM-UI-1. Paths are relative to the app base.
export const NAV: NavItem[] = [
  { href: '/', label: 'Dashboard', icon: 'dashboard' },
  {
    href: '/transactions',
    label: 'Transactions',
    icon: 'transactions',
    badge: 'pendingDrafts',
    children: [
      { href: '/transactions/drafts', label: 'Drafts', icon: 'inbox', badge: 'pendingDrafts' },
      { href: '/transactions/capture', label: 'Smart Capture', icon: 'camera' },
    ],
  },
  { href: '/budget', label: 'Budget', icon: 'budget' },
  {
    href: '/accounts',
    label: 'Accounts',
    icon: 'accounts',
    children: [
      { href: '/accounts/credit-cards', label: 'Credit Cards', icon: 'card' },
      { href: '/accounts/loans', label: 'Loans', icon: 'loan' },
      { href: '/accounts/goals', label: 'Goals', icon: 'goal' },
    ],
  },
  {
    href: '/investments',
    label: 'Investments',
    icon: 'investments',
    children: [
      { href: '/investments/market', label: 'Market Pulse', icon: 'activity' },
      { href: '/investments/analysis', label: 'Ticker Analysis', icon: 'search' },
      { href: '/investments/news', label: 'News Scanner', icon: 'newspaper' },
      { href: '/investments/think-tank', label: 'Think Tank', icon: 'sparkles' },
      { href: '/investments/funds', label: 'Fmarket', icon: 'layers' },
    ],
  },
  { href: '/health', label: 'Health', icon: 'health' },
  { href: '/chat', label: 'Assistant', icon: 'chat' },
  { href: '/settings', label: 'Settings', icon: 'settings' },
];

/** True when `path` (base stripped) is the item or one of its sub-pages. */
export function isActive(href: string, path: string): boolean {
  if (href === '/') return path === '/';
  return path === href || path.startsWith(`${href}/`);
}

/** Title of the most specific nav entry matching the path. */
export function titleFor(path: string): string {
  const flat = NAV.flatMap((item) => [item, ...(item.children ?? [])]);
  const match = flat.filter((item) => isActive(item.href, path)).sort((a, b) => b.href.length - a.href.length)[0];
  return match?.label ?? 'Wealth';
}
