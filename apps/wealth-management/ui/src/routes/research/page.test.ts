import { render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/paths', () => ({ base: '/ui' }));

import ResearchPage from './+page.svelte';

describe('Research area landing', () => {
  it('Given the research area When opening it Then it links every research tool with a description', () => {
    render(ResearchPage);

    const list = screen.getByRole('list', { name: 'Research tools' });
    const links = Array.from(list.querySelectorAll('a'));
    expect(links.map((link) => link.getAttribute('href'))).toEqual([
      '/ui/investments/market',
      '/ui/investments/analysis',
      '/ui/investments/news',
      '/ui/investments/think-tank',
      '/ui/investments/funds',
    ]);
    expect(screen.getByRole('link', { name: /Think Tank.*your holdings/ })).toBeTruthy();
    expect(screen.getByRole('heading', { level: 1, name: 'Research' })).toBeTruthy();
  });
});
