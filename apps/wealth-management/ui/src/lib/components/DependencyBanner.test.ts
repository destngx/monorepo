import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import DependencyBanner from './DependencyBanner.svelte';

describe('DependencyBanner', () => {
  it('stays quiet for an unconfigured cache and optional market providers', () => {
    const { container } = render(DependencyBanner, {
      health: {
        status: 'degraded',
        checks: [
          { name: 'sheets', status: 'ok' },
          { name: 'cache', status: 'unconfigured' },
          { name: 'market:vnstock', status: 'down', error: 'connection refused' },
        ],
      },
    });
    expect(container.textContent?.trim()).toBe('');
  });

  it('explains a missing sheet', () => {
    render(DependencyBanner, {
      health: { status: 'degraded', checks: [{ name: 'sheets', status: 'unconfigured' }] },
    });
    expect(screen.getByText(/Google Sheets is unavailable/)).toBeTruthy();
  });

  it('shows the setup steps for an unconfigured sheet', () => {
    render(DependencyBanner, {
      health: { status: 'degraded', checks: [{ name: 'sheets', status: 'unconfigured' }] },
    });
    expect(screen.getByRole('status').textContent).toMatch(/GOOGLE_SHEETS_ID/);
  });

  it('shows the re-auth steps when Google rejects the token', () => {
    render(DependencyBanner, {
      health: {
        status: 'down',
        checks: [
          { name: 'sheets', status: 'down', error: 'oauth2: "invalid_grant" "Token has been expired or revoked."' },
        ],
      },
    });
    const text = screen.getByRole('status').textContent ?? '';
    expect(text).toMatch(/auth:sheets/);
    expect(text).toMatch(/7 days/);
  });

  it('reports an unreachable engine', () => {
    render(DependencyBanner, { health: null });
    expect(screen.getByRole('alert').textContent).toMatch(/not reachable/);
  });
});
