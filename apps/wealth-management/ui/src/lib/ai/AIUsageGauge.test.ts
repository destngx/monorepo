import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from '$lib/api/endpoints';
import type { AIProviderUsage } from '$lib/api/types';
import AIUsageGauge from './AIUsageGauge.svelte';

function answer(usage: AIProviderUsage) {
  vi.spyOn(api, 'aiUsage').mockResolvedValue(usage);
}

describe('AIUsageGauge (WM-SET-4)', () => {
  afterEach(() => vi.restoreAllMocks());

  it('Given both windows, when the gauge is tapped, then the details open with a link to Settings', async () => {
    answer({
      provider: 'openai',
      windows: [
        { name: 'weekly', usedPercent: 60 },
        { name: '5h', usedPercent: 20 },
      ],
    });

    render(AIUsageGauge);

    const gauge = await screen.findByRole('button', {
      name: /^AI quota left: Weekly window 40\s?%, 5-hour window 80\s?%$/,
    });
    expect(gauge.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText(/40\s?% left/)).toBeNull();

    await fireEvent.click(gauge);

    expect(gauge.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText(/40\s?% left/)).toBeTruthy();
    expect(screen.getByText(/80\s?% left/)).toBeTruthy();
    expect(screen.getByRole('link', { name: 'AI settings' }).getAttribute('href')).toBe('/ui/settings');
  });

  it('Given the details are open, when Escape is pressed, the page is tapped elsewhere or the gauge again, then they close', async () => {
    answer({ provider: 'openai', windows: [{ name: 'weekly', usedPercent: 10 }] });

    render(AIUsageGauge);
    const gauge = await screen.findByRole('button', { name: /^AI quota left: Weekly window 90\s?%$/ });

    await fireEvent.click(gauge);
    expect(screen.getByText(/90\s?% left/)).toBeTruthy();
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(screen.queryByText(/90\s?% left/)).toBeNull();

    await fireEvent.click(gauge);
    await fireEvent.pointerDown(document.body);
    expect(screen.queryByText(/90\s?% left/)).toBeNull();

    await fireEvent.click(gauge);
    await fireEvent.click(gauge);
    expect(screen.queryByText(/90\s?% left/)).toBeNull();
  });

  it('Given the quota cannot be read, when tapped, then the details say why', async () => {
    vi.spyOn(api, 'aiUsage').mockRejectedValue(new Error('gateway down'));

    render(AIUsageGauge);
    const gauge = await screen.findByRole('button', { name: 'AI usage unavailable: gateway down' });

    await fireEvent.click(gauge);
    expect(screen.getByText('Unavailable: gateway down')).toBeTruthy();
  });
});
