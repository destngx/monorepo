import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import SourceBadge from './SourceBadge.svelte';

describe('SourceBadge', () => {
  it('names the provider quietly when the first provider answered', () => {
    render(SourceBadge, { source: { provider: 'fmarket', asOf: '2026-10-14T01:00:00Z' }, detail: 'bank sell' });
    const badge = screen.getByText('fmarket · bank sell');
    expect(badge.className).toContain('text-slate-600');
    expect(badge.getAttribute('title')).toMatch(/^Served by fmarket \(bank sell\), read /);
  });

  it('marks a fallback and lists what failed first', () => {
    render(SourceBadge, { source: { provider: 'yahoo', fallback: ['vnstock: quota'] } });
    const badge = screen.getByTitle(/Failed first: vnstock: quota/);
    expect(badge.className).toContain('text-amber-700');
    expect(badge.textContent).toContain('Fallback source:');
  });

  it('marks a stale copy with its age', () => {
    render(SourceBadge, { source: { provider: 'fmarket', stale: true, asOf: '2026-10-13T08:00:00Z' } });
    const badge = screen.getByTitle(/^Every provider failed/);
    expect(badge.className).toContain('text-orange-700');
    expect(badge.textContent).toMatch(/fmarket · as of \d{2}\/10\/2026/);
  });

  it('shows a dash without a source', () => {
    render(SourceBadge, {});
    expect(screen.getByText('-')).toBeTruthy();
  });
});
