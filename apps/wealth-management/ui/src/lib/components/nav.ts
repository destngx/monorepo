import type { IconName } from './Icon.svelte';

export interface NavItem {
  href: string;
  label: string;
  icon: IconName;
  children?: NavItem[];
}

// Navigation order from WM-UI-1. Paths are relative to the app base.
export const NAV: NavItem[] = [
  { href: '/', label: 'Dashboard', icon: 'dashboard' },
  { href: '/transactions', label: 'Transactions', icon: 'transactions' },
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
  { href: '/investments', label: 'Investments', icon: 'investments' },
  { href: '/health', label: 'Health', icon: 'health' },
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
