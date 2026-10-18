import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ apiTokenStatus: vi.fn(), setApiToken: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));

import { apiToken } from '$lib/state/settings.svelte';
import ApiTokenCard from './ApiTokenCard.svelte';

const oldToken = 'o'.repeat(32);

async function settle() {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
}

const field = () => screen.getByLabelText<HTMLInputElement>('API token');
const setOnEngine = () => screen.getByRole('button', { name: 'Set on engine' });

describe('ApiTokenCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiToken.value = '';
  });

  it('Given no token on localhost When generating and setting one Then the engine and this browser both get it', async () => {
    endpoints.apiTokenStatus.mockResolvedValue({ set: false, loopback: true });
    endpoints.setApiToken.mockImplementation(() => {
      // The request must still carry the browser's old token.
      expect(apiToken.value).toBe('');
      return Promise.resolve({ set: true, loopback: true });
    });
    render(ApiTokenCard);
    await settle();
    expect(screen.getByRole('status').textContent).toMatch(/only work from this Mac/);

    await fireEvent.click(screen.getByRole('button', { name: 'Generate' }));
    const generated = field().value;
    expect(generated).toMatch(/^[0-9a-f]{64}$/);
    expect(field().type).toBe('text');

    await fireEvent.click(setOnEngine());
    await settle();

    expect(endpoints.setApiToken).toHaveBeenCalledWith(generated);
    expect(apiToken.value).toBe(generated);
    expect(toastMock.success).toHaveBeenCalledWith('API token set on the engine and saved in this browser.');
    expect(screen.getByRole('status').textContent).toMatch(/requires a token/);
  });

  it('Given the engine refuses the token When setting it Then this browser keeps its old token', async () => {
    apiToken.value = oldToken;
    endpoints.apiTokenStatus.mockResolvedValue({ set: true, loopback: false });
    endpoints.setApiToken.mockRejectedValue(new Error('missing or invalid API token'));
    render(ApiTokenCard);
    await settle();

    await fireEvent.input(field(), { target: { value: 'n'.repeat(40) } });
    await fireEvent.click(setOnEngine());
    await settle();

    expect(apiToken.value).toBe(oldToken);
    expect(toastMock.success).not.toHaveBeenCalled();
  });

  it('Given a short token When typing Then it explains the rule and cannot be set', async () => {
    endpoints.apiTokenStatus.mockResolvedValue({ set: true, loopback: false });
    render(ApiTokenCard);
    await settle();

    await fireEvent.input(field(), { target: { value: 'short' } });

    expect(screen.getByText(/at least 32 characters/)).toBeTruthy();
    expect(setOnEngine()).toHaveProperty('disabled', true);
  });

  it('Given no token on the tailnet When opened Then it explains how to set the first one', async () => {
    endpoints.apiTokenStatus.mockResolvedValue({ set: false, loopback: false });
    render(ApiTokenCard);
    await settle();

    await fireEvent.click(screen.getByRole('button', { name: 'Generate' }));

    expect(screen.getByRole('status').textContent).toMatch(/refuses every change/);
    expect(setOnEngine()).toHaveProperty('disabled', true);
  });

  it('Given a token from another device When saving in this browser Then the engine is not touched', async () => {
    endpoints.apiTokenStatus.mockResolvedValue({ set: true, loopback: false });
    render(ApiTokenCard);
    await settle();

    await fireEvent.input(field(), { target: { value: ` ${oldToken} ` } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save in this browser' }));

    expect(apiToken.value).toBe(oldToken);
    expect(endpoints.setApiToken).not.toHaveBeenCalled();
    expect(toastMock.success).toHaveBeenCalledWith('API token saved in this browser.');
  });
});
