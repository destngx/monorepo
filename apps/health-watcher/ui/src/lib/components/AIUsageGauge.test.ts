import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { engine } from '$lib/api/endpoints';
import type { AIUsageView, Me } from '$lib/api/types';
import { session } from '$lib/state/session.svelte';
import AIUsageGauge from './AIUsageGauge.svelte';

function signIn(admin: boolean) {
  session.me = {
    user: { id: 'u1', email: 'user@example.test', displayName: 'User', admin, createdAt: '2026-10-22T00:00:00Z' },
    memberships: [],
    csrfToken: 't',
  } satisfies Me;
}

function answer(view: AIUsageView) {
  vi.spyOn(engine, 'aiUsage').mockResolvedValue(view);
}

describe('AIUsageGauge', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    session.me = null;
  });

  it('Given both windows, when an admin taps the gauge, then the details open with a link to the AI settings', async () => {
    signIn(true);
    answer({
      enabled: true,
      usage: {
        provider: 'openai',
        windows: [
          { name: '5h', usedPercent: 20 },
          { name: 'weekly', usedPercent: 60 },
        ],
      },
    });

    render(AIUsageGauge);

    const gauge = await screen.findByRole('button', { name: 'AI quota left: Weekly window 40%, 5-hour window 80%' });
    expect(gauge.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText('40% left')).toBeNull();

    await fireEvent.click(gauge);

    expect(gauge.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText('40% left')).toBeTruthy();
    expect(screen.getByText('80% left')).toBeTruthy();
    expect(screen.getByRole('link', { name: 'AI settings' }).getAttribute('href')).toBe('/ui/settings/engine');
  });

  it('Given a member who is not an admin, when the details open, then there is no settings link', async () => {
    signIn(false);
    answer({ enabled: true, usage: { provider: 'openai', windows: [{ name: 'weekly', usedPercent: 10 }] } });

    render(AIUsageGauge);
    await fireEvent.click(await screen.findByRole('button', { name: 'AI quota left: Weekly window 90%' }));

    expect(screen.getByText('90% left')).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'AI settings' })).toBeNull();
  });

  it('Given the gateway fails, when shown, then the gauge says why', async () => {
    signIn(true);
    answer({ enabled: true, error: 'the AI provider returned 502 Bad Gateway' });

    render(AIUsageGauge);

    expect(await screen.findByLabelText('AI usage unavailable: the AI provider returned 502 Bad Gateway')).toBeTruthy();
  });

  it('Given AI reading is off, when loaded, then nothing is shown', async () => {
    signIn(true);
    answer({ enabled: false });

    const { container } = render(AIUsageGauge);

    await waitFor(() => expect(engine.aiUsage).toHaveBeenCalled());
    expect(container.querySelector('svg')).toBeNull();
  });
});
