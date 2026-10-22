import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import AIUsageGauge from './AIUsageGauge.svelte';
import type { AIUsageSource } from './usage';

function renderGauge(state: Partial<AIUsageSource>, settingsHref?: string, rootClass?: string) {
  const load = vi.fn(() => Promise.resolve());
  const view = render(AIUsageGauge, {
    class: rootClass,
    source: { usage: null, error: '', loadedAt: new Date('2026-10-22T10:00:00Z'), load, ...state },
    leftColor: (left: number) => `color-${left}`,
    formatReset: (resetsAt: string) => `at ${resetsAt}`,
    errorClass: 'error',
    settingsHref,
  });
  return { load, ...view };
}

describe('AIUsageGauge', () => {
  it('Given both windows, when the gauge is tapped, then the details open with what is left, the resets and the settings link', async () => {
    const { load } = renderGauge(
      {
        usage: {
          provider: 'openai',
          windows: [
            { name: '5h', usedPercent: 20, resetsAt: '2026-10-22T11:30:00Z' },
            { name: 'weekly', usedPercent: 60 },
          ],
        },
      },
      '/ui/settings',
    );

    expect(load).toHaveBeenCalledOnce();
    const gauge = screen.getByRole('button', { name: 'AI quota left: Weekly window 40%, 5-hour window 80%' });
    expect(gauge.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText('40% left')).toBeNull();

    await fireEvent.click(gauge);

    expect(gauge.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText('AI quota left (openai)')).toBeTruthy();
    expect(screen.getByText('40% left')).toBeTruthy();
    expect(screen.getByText('80% left')).toBeTruthy();
    expect(screen.getByText('at 2026-10-22T11:30:00Z')).toBeTruthy();
    expect(screen.getByText(/\(in 1h 30m\)/)).toBeTruthy();
    expect(screen.getByRole('link', { name: 'AI settings' }).getAttribute('href')).toBe('/ui/settings');
  });

  it('Given the details are open, when Escape is pressed, the page is tapped elsewhere or the gauge again, then they close', async () => {
    renderGauge({ usage: { provider: 'openai', windows: [{ name: 'weekly', usedPercent: 10 }] } });
    const gauge = screen.getByRole('button', { name: 'AI quota left: Weekly window 90%' });

    await fireEvent.click(gauge);
    expect(screen.getByText('90% left')).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'AI settings' })).toBeNull();
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(screen.queryByText('90% left')).toBeNull();

    await fireEvent.click(gauge);
    await fireEvent.pointerDown(document.body);
    expect(screen.queryByText('90% left')).toBeNull();

    await fireEvent.click(gauge);
    await fireEvent.click(gauge);
    expect(screen.queryByText('90% left')).toBeNull();
  });

  it('Given the quota cannot be read, when tapped, then the details say why', async () => {
    renderGauge({ error: 'gateway down' });
    const gauge = screen.getByRole('button', { name: 'AI usage unavailable: gateway down' });

    await fireEvent.click(gauge);
    expect(screen.getByText('Unavailable: gateway down').className).toBe('error');
  });

  it('Given only token totals, when shown, then there is nothing to gauge', () => {
    const { container } = renderGauge({ usage: { provider: 'openai', windows: [] } });
    expect(container.querySelector('svg')).toBeNull();
  });

  it('Given AI is switched off, when shown, then the gauge hides even with an error', () => {
    const { container } = renderGauge({ enabled: false, error: 'gateway down' });
    expect(container.querySelector('svg')).toBeNull();
  });

  it('Given an app anchor class, when shown, then the details are positioned against the wrapper the app chose', () => {
    const weekly = { name: 'weekly', usedPercent: 10 };
    const anchored = renderGauge({ usage: { provider: 'openai', windows: [weekly] } }, undefined, 'sm:relative');
    expect(anchored.container.firstElementChild?.className).toBe('sm:relative');
    anchored.unmount();

    const plain = renderGauge({ usage: { provider: 'openai', windows: [weekly] } });
    expect(plain.container.firstElementChild?.className).toBe('relative');
  });
});
