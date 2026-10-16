import { describe, expect, it } from 'vitest';
import { AREAS, PRIMARY_AREAS, areaFor, areaPages, isPageActive, pageFor, titleFor } from './nav';

// Every page route of the app, as a path with sample values for parameters.
const routeFiles = Object.keys(import.meta.glob('/src/routes/**/+page.svelte'));
const routePaths = routeFiles.map((file) => {
  const path = file
    .replace(/^\/src\/routes/, '')
    .replace(/\/\+page\.svelte$/, '')
    .replace(/\[[^\]]+\]/g, 'sample');
  return path || '/';
});

describe('five-area navigation', () => {
  it('Given every page route When resolving its area Then each lands in exactly one area', () => {
    expect(routePaths.length).toBeGreaterThan(20);
    for (const path of routePaths) {
      const area = areaFor(path);
      expect(area, path).toBeDefined();
      const claiming = AREAS.filter((candidate) =>
        areaPages(candidate).some((page) => page.href === pageFor(path)?.href),
      );
      expect(
        claiming.map((candidate) => candidate.id),
        path,
      ).toEqual([area?.id]);
    }
  });

  it('Given the nav model When listing its pages Then every page is a real route, listed once', () => {
    const hrefs = AREAS.flatMap((area) => areaPages(area).map((page) => page.href));
    expect(new Set(hrefs).size).toBe(hrefs.length);
    for (const href of hrefs) expect(routePaths, href).toContain(href);
    for (const area of AREAS) expect(hrefs, area.id).toContain(area.href);
  });

  it('Given the bar When listing the primary areas Then it is Home, Money, Wealth, Plan, Research and Assistant', () => {
    expect(PRIMARY_AREAS.map((area) => area.label)).toEqual([
      'Home',
      'Money',
      'Wealth',
      'Plan',
      'Research',
      'Assistant',
    ]);
  });

  it.each([
    ['/', 'home'],
    ['/transactions/drafts', 'money'],
    ['/transactions/capture', 'money'],
    ['/budget', 'money'],
    ['/cash-flow', 'money'],
    ['/bills', 'money'],
    ['/accounts/credit-cards/Card%20A', 'money'],
    ['/accounts', 'wealth'],
    ['/accounts/lent', 'wealth'],
    ['/investments', 'wealth'],
    ['/accounts/goals', 'plan'],
    ['/health', 'plan'],
    ['/close', 'plan'],
    ['/recommendations', 'plan'],
    ['/research', 'research'],
    ['/investments/market', 'research'],
    ['/investments/analysis', 'research'],
    ['/investments/news', 'research'],
    ['/investments/think-tank', 'research'],
    ['/investments/funds/FUND-A', 'research'],
    ['/chat', 'assistant'],
    ['/settings', 'settings'],
  ])('Given %s When resolving the area Then it is %s', (path, area) => {
    expect(areaFor(path)?.id).toBe(area);
  });

  it('Given a nested route When marking pages active Then only the most specific branch is active', () => {
    expect(isPageActive('/transactions', '/transactions/drafts')).toBe(true);
    expect(isPageActive('/transactions/drafts', '/transactions/drafts')).toBe(true);
    // Credit cards and goals live under /accounts but belong to other areas.
    expect(isPageActive('/accounts', '/accounts/credit-cards/Card')).toBe(false);
    expect(isPageActive('/accounts', '/accounts/goals')).toBe(false);
    expect(isPageActive('/accounts/credit-cards', '/accounts/credit-cards/Card')).toBe(true);
    // Research tools live under /investments but are not Investments.
    expect(isPageActive('/investments', '/investments/think-tank')).toBe(false);
    expect(isPageActive('/investments', '/investments')).toBe(true);
    expect(isPageActive('/', '/accounts')).toBe(false);
    expect(isPageActive('/accounts', '/accountsx')).toBe(false);
  });

  it('Given a path When titling the page Then the most specific entry names it', () => {
    expect(titleFor('/accounts/credit-cards/Visa')).toBe('Credit Cards');
    expect(titleFor('/')).toBe('Home');
    expect(titleFor('/bills')).toBe('Bills');
    expect(titleFor('/recommendations')).toBe('Recommendations');
    expect(titleFor('/close')).toBe('Monthly close');
    expect(titleFor('/research')).toBe('Research');
    expect(titleFor('/nowhere')).toBe('Wealth');
  });
});
