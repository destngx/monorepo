import type { EngineSettings } from '$lib/api/types';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({
  settings: vi.fn(),
  updateSettings: vi.fn(),
  restartEngine: vi.fn(),
}));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
const navigation = vi.hoisted(() => ({
  invalidateAll: vi.fn(() => Promise.resolve()),
  goto: vi.fn(() => Promise.resolve()),
}));
const appState = vi.hoisted(() => ({ page: { url: new URL('http://127.0.0.1:5173/ui/settings') } }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn() }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/state', () => appState);
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));

import EngineSettingsEditor from './EngineSettings.svelte';

const view = (restartRequired = false): EngineSettings => ({
  groups: [{ id: 'ai', label: 'AI', description: 'Provider' }],
  settings: [
    {
      key: 'AI_MODEL',
      group: 'ai',
      label: 'Default model',
      description: '',
      kind: 'text',
      secret: false,
      isSet: true,
      value: 'gpt',
    },
    { key: 'AI_API_KEY', group: 'ai', label: 'API key', description: '', kind: 'secret', secret: true, isSet: true },
    {
      key: 'LOG_LEVEL',
      group: 'ai',
      label: 'Log level',
      description: '',
      kind: 'choice',
      default: 'info',
      options: ['debug', 'info'],
      secret: false,
      isSet: false,
    },
  ],
  path: '/config/settings.db',
  restartRequired,
});

async function settle() {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
}

describe('EngineSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.settings.mockResolvedValue(view());
    appState.page.url = new URL('http://127.0.0.1:5173/ui/settings');
  });

  it('reports a Google sign-in the engine redirected back with, then drops it from the address', async () => {
    appState.page.url = new URL('http://127.0.0.1:5173/ui/settings?tab=x&google=connected&purpose=sheets');
    render(EngineSettingsEditor);
    await settle();

    expect(toastMock.push).toHaveBeenCalledWith('success', 'Google account connected. Restart the engine to use it.');
    expect(navigation.goto).toHaveBeenCalledWith(new URL('http://127.0.0.1:5173/ui/settings?tab=x'), {
      replaceState: true,
      noScroll: true,
      keepFocus: true,
    });
  });

  it('leaves the address alone on a normal visit', async () => {
    render(EngineSettingsEditor);
    await settle();

    expect(navigation.goto).not.toHaveBeenCalled();
  });

  it('offers Connect Google account on the Sheets card, blocked while changes are unsaved', async () => {
    const sheets = view();
    sheets.groups.push({ id: 'sheets', label: 'Google Sheets', description: '' });
    sheets.settings.push({
      key: 'GOOGLE_SHEETS_ID',
      group: 'sheets',
      label: 'Spreadsheet ID',
      description: '',
      kind: 'text',
      secret: false,
      isSet: false,
    });
    endpoints.settings.mockResolvedValue(sheets);
    render(EngineSettingsEditor);
    await settle();

    const connect = screen.getByRole<HTMLButtonElement>('button', { name: 'Connect Google account' });
    expect(connect.disabled).toBe(false);
    expect(screen.queryByRole('button', { name: 'Connect test account' })).toBeNull();

    await fireEvent.input(screen.getByLabelText('Spreadsheet ID'), { target: { value: 'sheet-1' } });
    expect(connect.disabled).toBe(true);
  });

  it('shows stored values but never a secret value', async () => {
    render(EngineSettingsEditor);
    await settle();

    expect(screen.getByLabelText<HTMLInputElement>('Default model').value).toBe('gpt');
    const secret = screen.getByLabelText<HTMLInputElement>('API key');
    expect(secret.type).toBe('password');
    expect(secret.value).toBe('');
    expect(screen.getByText('set')).toBeTruthy();
    expect(screen.getByText('/config/settings.db')).toBeTruthy();
  });

  it('saves only the changed settings and then offers the restart', async () => {
    endpoints.updateSettings.mockResolvedValue(view(true));
    render(EngineSettingsEditor);
    await settle();

    await fireEvent.input(screen.getByLabelText('Default model'), { target: { value: 'claude' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }));
    await settle();

    expect(endpoints.updateSettings).toHaveBeenCalledWith({ AI_MODEL: 'claude', AI_API_KEY: null });
    expect(screen.getByText('Restart to apply')).toBeTruthy();
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Restart engine' }).disabled).toBe(false);
  });

  it('shows the engine validation error', async () => {
    endpoints.updateSettings.mockRejectedValue(
      new Error('validation failed: AI_TIMEOUT_SECONDS must be a positive integer'),
    );
    render(EngineSettingsEditor);
    await settle();

    await fireEvent.input(screen.getByLabelText('Default model'), { target: { value: 'claude' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }));
    await settle();

    expect(screen.getByRole('alert').textContent).toMatch(/AI_TIMEOUT_SECONDS/);
  });

  it('blocks the restart while changes are unsaved', async () => {
    render(EngineSettingsEditor);
    await settle();

    await fireEvent.input(screen.getByLabelText('Default model'), { target: { value: 'claude' } });

    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Restart engine' }).disabled).toBe(true);
  });
});
