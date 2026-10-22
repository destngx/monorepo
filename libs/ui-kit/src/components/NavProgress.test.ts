import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import NavProgress from './NavProgress.svelte';
import { NAV_PROGRESS_DELAY_MS, NAV_PROGRESS_FADE_MS } from './navProgress';

const bar = () => screen.queryByRole('progressbar', { name: 'Loading page' });
const phase = (container: HTMLElement) => container.querySelector('[data-nav-progress]')?.getAttribute('data-phase');

async function advance(ms: number) {
  vi.advanceTimersByTime(ms);
  await tick();
}

describe('NavProgress', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('Given a quick navigation When it ends before the delay Then the bar never shows', async () => {
    const { container, rerender } = render(NavProgress, { active: false });

    await rerender({ active: true });
    await advance(NAV_PROGRESS_DELAY_MS - 1);
    await rerender({ active: false });
    await advance(NAV_PROGRESS_DELAY_MS + NAV_PROGRESS_FADE_MS);

    expect(bar()).toBeNull();
    expect(phase(container)).toBe('idle');
  });

  it('Given a slow navigation When the delay passes Then the bar creeps until the page arrives, then fills and fades', async () => {
    const { container, rerender } = render(NavProgress, { active: false });

    await rerender({ active: true });
    await advance(NAV_PROGRESS_DELAY_MS);
    expect(bar()).toBeTruthy();
    expect(phase(container)).toBe('loading');

    await rerender({ active: false });
    expect(phase(container)).toBe('done');
    expect(bar()).toBeNull();

    await advance(NAV_PROGRESS_FADE_MS);
    expect(phase(container)).toBe('idle');
  });

  it('Given the bar fading out When another navigation starts Then it restarts from empty', async () => {
    const { container, rerender } = render(NavProgress, { active: true });
    await advance(NAV_PROGRESS_DELAY_MS);
    await rerender({ active: false });

    await rerender({ active: true });
    expect(phase(container)).toBe('idle');
    await advance(NAV_PROGRESS_DELAY_MS);
    expect(phase(container)).toBe('loading');
  });
});
