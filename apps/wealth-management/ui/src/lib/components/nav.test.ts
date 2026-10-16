import { describe, expect, it } from 'vitest';
import { isActive, titleFor } from './nav';

describe('nav', () => {
  it('marks parents active on sub-pages but not the dashboard', () => {
    expect(isActive('/accounts', '/accounts/lent')).toBe(true);
    expect(isActive('/', '/accounts')).toBe(false);
    expect(isActive('/accounts', '/accountsx')).toBe(false);
  });

  it('titles a page by its most specific entry', () => {
    expect(titleFor('/accounts/credit-cards/Visa')).toBe('Credit Cards');
    expect(titleFor('/')).toBe('Dashboard');
    expect(titleFor('/bills')).toBe('Bills');
    expect(titleFor('/recommendations')).toBe('Recommendations');
  });
});
