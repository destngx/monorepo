import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { AIStatus } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import CapturePage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};
const pending = () => ({ engine: never(), options: never(), aiStatus: never() });

// Invented status.
const ready: AIStatus = {
  configured: true,
  provider: 'openai',
  endpoint: 'ai.example.test',
  models: {},
  fallbackModel: '',
  prompts: 3,
  missingPrompts: [],
  capture: { maxItems: 20, maxItemBytes: 5 * 1024 * 1024, maxBatchBytes: 20 * 1024 * 1024 },
};

async function addText(text: string) {
  await fireEvent.input(screen.getByLabelText('Text material'), { target: { value: text } });
  await fireEvent.click(screen.getByRole('button', { name: 'Add text' }));
  await flush();
}

describe('Smart Capture page', () => {
  afterEach(() => mask.hide());

  it('Given nothing has answered When the page renders Then materials can be added and Analyze waits for the AI status', async () => {
    render(CapturePage, { data: pending() as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Smart Capture' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Choose images' }).hasAttribute('disabled')).toBe(false);
    await addText('Paid 85.000 VND at Noodle Bar');

    const analyze = screen.getByRole('button', { name: /Analyze/ });
    expect(analyze.getAttribute('aria-busy')).toBe('true');
    expect(analyze.getAttribute('title')).toBe('Checking that AI is available…');
  });

  it('Given the AI status fails When it settles Then it shows the error with Retry and Analyze stays off', async () => {
    render(CapturePage, {
      data: { ...pending(), aiStatus: defer(Promise.reject(new Error('engine broke'))) } as never,
    });
    await flush();
    await addText('Paid 85.000 VND at Noodle Bar');

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('the AI status');
    expect(alert.textContent).toContain('engine broke');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('button', { name: /Analyze/ }).hasAttribute('disabled')).toBe(true);
  });

  it('Given AI is ready and a material is added When the status loads Then Analyze is enabled', async () => {
    render(CapturePage, { data: { ...pending(), aiStatus: settled(ready) } as never });
    await flush();
    await addText('Paid 85.000 VND at Noodle Bar');

    const analyze = screen.getByRole('button', { name: 'Analyze 1' });
    expect(analyze.hasAttribute('disabled')).toBe(false);
    expect(analyze.getAttribute('aria-busy')).toBeNull();
    expect(screen.getByText(/ai\.example\.test/)).toBeTruthy();
  });

  it('Given a text material quoting an amount When masked Then its preview hides the amount until toggled', async () => {
    render(CapturePage, { data: pending() as never });
    await addText('Paid 85.000 VND at Noodle Bar');

    const preview = screen.getByRole('list', { name: 'Materials' });
    expect(preview.textContent).toContain(`Paid ${MASK} at Noodle Bar`);
    expect(preview.textContent).not.toContain('85.000');

    mask.toggle();
    await flush();
    expect(preview.textContent).toContain('Paid 85.000 VND at Noodle Bar');
  });
});
