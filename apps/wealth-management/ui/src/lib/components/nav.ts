import { type IconName } from '@libs/ui-kit';

// Five-area navigation (ROADMAP P1, WM-UI-10..13). Every page route belongs
// to exactly one area; URLs are unchanged, so an area's pages may sit under
// another area's path (credit cards and goals under /accounts, the research
// tools under /investments). Active state is therefore resolved by the most
// specific page, never by path prefix alone. Paths are relative to the app base.

export interface NavPage {
  href: string;
  label: string;
  icon: IconName;
  /** Header title when it differs from the label. */
  title?: string;
  /** One line on what the page answers (area landing pages). */
  description?: string;
  children?: NavPage[];
  /** A count shown next to the label, e.g. drafts waiting for review. */
  badge?: 'pendingDrafts';
}

export type AreaId = 'home' | 'money' | 'wealth' | 'plan' | 'research' | 'assistant' | 'settings';

export interface NavArea {
  id: AreaId;
  label: string;
  icon: IconName;
  /** Where the area's bar entry goes. */
  href: string;
  pages: NavPage[];
  /** Utility areas (Settings) stay out of the area bar. */
  utility?: boolean;
  badge?: 'pendingDrafts';
}

export const AREAS: NavArea[] = [
  {
    id: 'home',
    label: 'Home',
    icon: 'home',
    href: '/',
    pages: [{ href: '/', label: 'Home', icon: 'home' }],
  },
  {
    id: 'money',
    label: 'Money',
    icon: 'wallet',
    href: '/transactions',
    badge: 'pendingDrafts',
    pages: [
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
      { href: '/cash-flow', label: 'Cash Flow', icon: 'calendar' },
      { href: '/bills', label: 'Bills', icon: 'clock' },
      { href: '/accounts/credit-cards', label: 'Credit Cards', icon: 'card' },
    ],
  },
  {
    id: 'wealth',
    label: 'Wealth',
    icon: 'accounts',
    href: '/accounts',
    pages: [
      { href: '/accounts', label: 'Accounts', icon: 'accounts' },
      { href: '/investments', label: 'Investments', icon: 'investments' },
      { href: '/accounts/lent', label: 'Money lent', icon: 'loan' },
    ],
  },
  {
    id: 'plan',
    label: 'Plan',
    icon: 'goal',
    href: '/accounts/goals',
    pages: [
      { href: '/accounts/goals', label: 'Goals', icon: 'goal' },
      { href: '/health', label: 'Health', icon: 'health' },
      { href: '/recommendations', label: 'Recommendations', icon: 'lightbulb' },
      { href: '/close', label: 'Monthly close', icon: 'check' },
    ],
  },
  {
    id: 'research',
    label: 'Research',
    icon: 'compass',
    href: '/research',
    pages: [
      { href: '/research', label: 'Overview', title: 'Research', icon: 'compass' },
      {
        href: '/investments/market',
        label: 'Market Pulse',
        icon: 'activity',
        description: 'How the Vietnamese market is moving: regime, movers, correlations and capital flow.',
      },
      {
        href: '/investments/analysis',
        label: 'Ticker Analysis',
        icon: 'search',
        description: 'One symbol in depth: technicals across timeframes, seasonality and an AI read.',
      },
      {
        href: '/investments/news',
        label: 'News Scanner',
        icon: 'newspaper',
        description: 'Scan news topics and see whether the coverage leans bullish or bearish.',
      },
      {
        href: '/investments/think-tank',
        label: 'Think Tank',
        icon: 'sparkles',
        description:
          'An AI debate over your holdings, or symbols you choose; what is worth doing goes to your recommendations.',
      },
      {
        href: '/investments/funds',
        label: 'Fmarket',
        icon: 'layers',
        description: 'Open-ended funds, published bank deposit rates, gold and USD.',
      },
      {
        href: '/investments/stablecoin',
        label: 'Stablecoin Switch',
        icon: 'calculator',
        description: 'Whether a higher stablecoin rate elsewhere is worth the conversion fees, and how long to hold.',
      },
    ],
  },
  {
    id: 'assistant',
    label: 'Assistant',
    icon: 'chat',
    href: '/chat',
    pages: [{ href: '/chat', label: 'Assistant', icon: 'chat' }],
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: 'settings',
    href: '/settings',
    utility: true,
    pages: [{ href: '/settings', label: 'Settings', icon: 'settings' }],
  },
];

/** The area bar: the five areas plus Assistant. */
export const PRIMARY_AREAS = AREAS.filter((area) => !area.utility);

/** An area's pages with their sub-pages, flattened. */
export function areaPages(area: NavArea): NavPage[] {
  return area.pages.flatMap((page) => [page, ...(page.children ?? [])]);
}

const ALL = AREAS.flatMap((area) => areaPages(area).map((page) => ({ area, page })));

function matches(href: string, path: string): boolean {
  if (href === '/') return path === '/';
  return path === href || path.startsWith(`${href}/`);
}

function resolve(path: string) {
  return ALL.filter(({ page }) => matches(page.href, path)).sort((a, b) => b.page.href.length - a.page.href.length)[0];
}

/** The most specific nav page for a path (base stripped). */
export function pageFor(path: string): NavPage | undefined {
  return resolve(path)?.page;
}

/** The area a path belongs to. */
export function areaFor(path: string): NavArea | undefined {
  return resolve(path)?.area;
}

/**
 * True when the page is the one shown, or the parent of the one shown
 * (Transactions on Drafts). Pages of other areas under the same path never
 * light up their path parent (Accounts on Credit Cards).
 */
export function isPageActive(href: string, path: string): boolean {
  const current = pageFor(path);
  if (!current) return false;
  if (current.href === href) return true;
  const parent = ALL.find(({ page }) => page.href === href)?.page;
  return parent?.children?.some((child) => child.href === current.href) ?? false;
}

/** Header title: the most specific entry, or the app name. */
export function titleFor(path: string): string {
  const page = pageFor(path);
  return page ? (page.title ?? page.label) : 'Wealth';
}
