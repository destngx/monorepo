import { render, screen, within } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const appState = vi.hoisted(() => ({ page: { url: new URL('http://127.0.0.1/ui/') } }));
const drafts = vi.hoisted(() => ({ pendingDrafts: { count: 0 } }));
vi.mock('$app/state', () => appState);
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$lib/state/drafts.svelte', () => drafts);

import AreaTabs from './AreaTabs.svelte';
import MobileNav from './MobileNav.svelte';
import Sidebar from './Sidebar.svelte';

function at(path: string) {
  appState.page.url = new URL(`http://127.0.0.1/ui${path}`);
}

beforeEach(() => {
  drafts.pendingDrafts.count = 0;
});

describe('Sidebar', () => {
  it('Given the desktop sidebar When it renders Then the pages are grouped under labelled areas', () => {
    at('/');
    render(Sidebar);

    const nav = screen.getByRole('navigation', { name: 'Main' });
    for (const area of ['Money', 'Wealth', 'Plan', 'Research']) {
      const group = within(nav).getByRole('list', { name: area });
      expect(group).toBeTruthy();
    }
    const money = within(nav).getByRole('list', { name: 'Money' });
    expect(within(money).getByRole('link', { name: 'Credit Cards' }).getAttribute('href')).toBe(
      '/ui/accounts/credit-cards',
    );
    const research = within(nav).getByRole('list', { name: 'Research' });
    expect(within(research).getByRole('link', { name: 'Think Tank' })).toBeTruthy();
    expect(within(nav).getByRole('link', { name: 'Home' }).getAttribute('aria-current')).toBe('page');
    expect(within(nav).getByRole('link', { name: 'Settings' })).toBeTruthy();
  });

  it('Given a credit card page When the sidebar renders Then Credit Cards is current, not Accounts', () => {
    at('/accounts/credit-cards/Card%20A');
    render(Sidebar);

    expect(screen.getByRole('link', { name: 'Credit Cards' }).getAttribute('aria-current')).toBe('page');
    expect(screen.getByRole('link', { name: 'Accounts' }).getAttribute('aria-current')).toBeNull();
  });

  it('Given drafts are waiting When on their page Then Transactions opens its sub-pages and shows the count', () => {
    drafts.pendingDrafts.count = 3;
    at('/transactions/drafts');
    render(Sidebar);

    expect(screen.getByRole('link', { name: /^Drafts/ }).getAttribute('aria-current')).toBe('page');
    expect(screen.getByRole('link', { name: /^Drafts/ }).textContent).toContain('3');
    expect(screen.getByRole('link', { name: 'Smart Capture' })).toBeTruthy();
  });

  it('Given the sidebar is collapsed When it renders Then each area is one labelled icon link', () => {
    at('/investments/news');
    render(Sidebar, { collapsed: true });

    const links = screen.getAllByRole('link');
    expect(links.map((link) => link.getAttribute('aria-label'))).toEqual([
      'Home',
      'Money',
      'Wealth',
      'Plan',
      'Research',
      'Assistant',
      'Settings',
    ]);
    expect(screen.getByRole('link', { name: 'Research' }).getAttribute('aria-current')).toBe('true');
  });
});

describe('MobileNav', () => {
  it('Given a phone When on a research tool Then the area bar marks Research with a visible label', () => {
    at('/investments/think-tank');
    render(MobileNav);

    const bar = screen.getByRole('navigation', { name: 'Areas' });
    const links = within(bar).getAllByRole('link');
    expect(links.map((link) => link.textContent?.trim())).toEqual([
      'Home',
      'Money',
      'Wealth',
      'Plan',
      'Research',
      'Assistant',
    ]);
    expect(within(bar).getByRole('link', { name: 'Research' }).getAttribute('aria-current')).toBe('true');
    expect(within(bar).getByRole('link', { name: 'Home' }).getAttribute('aria-current')).toBeNull();
  });

  it('Given drafts are waiting When the bar renders Then Money says so in text', () => {
    drafts.pendingDrafts.count = 2;
    at('/');
    render(MobileNav);

    expect(screen.getByRole('link', { name: /Money, 2 drafts waiting/ })).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Home' }).getAttribute('aria-current')).toBe('page');
  });
});

describe('AreaTabs', () => {
  it('Given a page of a multi-page area When it renders Then the area pages are tabs with the current one marked', () => {
    at('/health');
    render(AreaTabs);

    const nav = screen.getByRole('navigation', { name: 'Plan pages' });
    expect(
      within(nav)
        .getAllByRole('link')
        .map((link) => link.textContent?.trim()),
    ).toEqual(['Goals', 'Health', 'Recommendations', 'Monthly close']);
    expect(within(nav).getByRole('link', { name: 'Health' }).getAttribute('aria-current')).toBe('page');
  });

  it('Given a single-page area When it renders Then there are no tabs', () => {
    at('/chat');
    render(AreaTabs);

    expect(screen.queryByRole('navigation')).toBeNull();
  });
});
