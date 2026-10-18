import type { EngineSetting, EngineSettings } from '$lib/api/types';
import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({
  settings: vi.fn(),
  updateSettings: vi.fn(),
  restartEngine: vi.fn(),
  aiModels: vi.fn(),
  aiStatus: vi.fn(),
  preferences: vi.fn(),
  aiUsage: vi.fn(),
  sheetInfo: vi.fn(),
  startGoogleAuth: vi.fn(),
  apiTokenStatus: vi.fn(),
}));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
const navigation = vi.hoisted(() => ({
  invalidateAll: vi.fn(() => Promise.resolve()),
  goto: vi.fn(() => Promise.resolve()),
}));
const appState = vi.hoisted(() => ({
  page: { url: new URL('http://127.0.0.1:5173/ui/settings'), data: { health: null } },
}));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/state', () => appState);
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
// The restarted engine answers at once instead of after a poll interval.
vi.mock('$lib/settings/form', async (original) => ({
  ...(await original<typeof import('$lib/settings/form')>()),
  waitForRestart: (load: () => Promise<EngineSettings>) => load(),
}));

import SettingsPage from './+page.svelte';

const setting = (key: string, group: string, label: string, extra: Partial<EngineSetting> = {}): EngineSetting => ({
  key,
  group,
  label,
  description: '',
  kind: 'text',
  secret: false,
  isSet: false,
  ...extra,
});

const view = (restartRequired = false): EngineSettings => ({
  groups: [
    { id: 'sheets', label: 'Google Sheets', description: '' },
    { id: 'ai', label: 'AI provider', description: '' },
    { id: 'search', label: 'Web search', description: '' },
    { id: 'market', label: 'Market data', description: '' },
    { id: 'general', label: 'Engine', description: '' },
  ],
  settings: [
    setting('GOOGLE_SHEETS_ID', 'sheets', 'Spreadsheet ID'),
    setting('AI_API_KEY', 'ai', 'API key', { kind: 'secret', secret: true, isSet: true }),
    setting('AI_MODEL', 'ai', 'Default model', { kind: 'model', isSet: true, value: 'gpt-5' }),
    setting('AI_MODEL_CHAT', 'ai', 'Chat model', { kind: 'model', inherits: 'AI_MODEL' }),
    setting('AI_TIMEOUT_SECONDS', 'ai', 'Timeout (seconds)', { kind: 'integer', default: '90', advanced: true }),
    setting('TAVILY_API_KEY', 'search', 'Tavily API key', { kind: 'secret', secret: true }),
    setting('MARKET_DATA_SERVER_URL', 'market', 'Market data server URL', {
      kind: 'url',
      default: 'http://localhost:8000',
    }),
    setting('LOG_LEVEL', 'general', 'Log level', { kind: 'choice', default: 'info', options: ['debug', 'info'] }),
  ],
  path: '/config/settings.db',
  restartRequired,
});

async function settle() {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
}

async function open(tab = '') {
  appState.page.url = new URL(`http://127.0.0.1:5173/ui/settings${tab ? `?tab=${tab}` : ''}`);
  render(SettingsPage);
  await settle();
}

describe('Settings page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.settings.mockResolvedValue(view());
    endpoints.apiTokenStatus.mockResolvedValue({ set: true, loopback: false });
    endpoints.aiModels.mockResolvedValue({ models: ['claude-opus-5-5', 'gpt-5'] });
    endpoints.aiStatus.mockResolvedValue({
      configured: true,
      models: {},
      prompts: 3,
      missingPrompts: [],
      missingKeys: [],
    });
    endpoints.aiUsage.mockRejectedValue(new Error('no usage'));
    endpoints.sheetInfo.mockResolvedValue({ spreadsheetId: 'sheet-1', title: 'Money' });
  });

  it('opens on General with the browser preferences and the engine card', async () => {
    await open();

    expect(screen.getByRole('link', { name: 'General' }).getAttribute('aria-current')).toBe('page');
    expect(screen.getByText('This browser')).toBeTruthy();
    const logLevel = screen.getByLabelText<HTMLSelectElement>('Log level');
    expect([...logLevel.options].map((option) => option.textContent)).toEqual(['Default (info)', 'debug', 'info']);
    expect(screen.getByText('/config/settings.db')).toBeTruthy();
    expect(screen.queryByLabelText('Market data server URL')).toBeNull();
  });

  it('shows only the tab in the address', async () => {
    await open('market');

    expect(screen.getByLabelText<HTMLInputElement>('Market data server URL').placeholder).toBe(
      'Default (http://localhost:8000)',
    );
    expect(screen.queryByText('This browser')).toBeNull();
  });

  it('folds rarely changed settings under a closed Advanced section', async () => {
    await open('ai');

    const advanced = screen.getByText('Advanced (1)').closest('details');
    expect(advanced?.open).toBe(false);
    expect(within(advanced as HTMLElement).getByLabelText('Timeout (seconds)')).toBeTruthy();
  });

  it('keeps the env key out of the text, in the label tooltip', async () => {
    await open('market');

    expect(screen.queryByText(/MARKET_DATA_SERVER_URL/)).toBeNull();
    expect(screen.getByText('Market data server URL').getAttribute('title')).toBe('MARKET_DATA_SERVER_URL');
  });

  it('offers the endpoint models as options instead of free text', async () => {
    await open('ai');

    const model = screen.getByLabelText<HTMLSelectElement>('Default model');
    expect(model.tagName).toBe('SELECT');
    expect(model.value).toBe('gpt-5');
    expect([...model.options].map((option) => option.value)).toEqual(['', 'claude-opus-5-5', 'gpt-5']);
    const chat = screen.getByLabelText<HTMLSelectElement>('Chat model');
    expect(chat.options[0].textContent).toBe('Default (gpt-5)');
  });

  it('falls back to typing model ids when the endpoint cannot list them', async () => {
    endpoints.aiModels.mockRejectedValue(new Error('ai unavailable: set AI_BASE_URL'));
    await open('ai');

    expect(screen.getByLabelText('Default model').tagName).toBe('INPUT');
    expect(screen.getByText(/Could not list the endpoint's models/)).toBeTruthy();
  });

  it('never shows a secret value', async () => {
    await open('ai');

    const secret = screen.getByLabelText<HTMLInputElement>('API key');
    expect(secret.type).toBe('password');
    expect(secret.value).toBe('');
    expect(screen.getByText('stored')).toBeTruthy();
  });

  it('saves only the changed settings, marks the tab meanwhile, then offers the restart', async () => {
    endpoints.updateSettings.mockResolvedValue(view(true));
    await open('ai');

    await fireEvent.change(screen.getByLabelText('Default model'), { target: { value: 'claude-opus-5-5' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    expect(screen.getByText('2 unsaved changes')).toBeTruthy();
    expect(within(screen.getByRole('link', { name: /^AI/ })).getByText('(unsaved changes)')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }));
    await settle();

    expect(endpoints.updateSettings).toHaveBeenCalledWith({ AI_MODEL: 'claude-opus-5-5', AI_API_KEY: null });
    expect(screen.getByText('Saved. Restart the engine to apply.')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Restart engine' })).toBeTruthy();
  });

  it('shows what the restarted engine runs with: AI status and the new endpoint models', async () => {
    const status = { configured: true, models: {}, prompts: 3, missingPrompts: [], missingKeys: [] };
    endpoints.settings.mockResolvedValueOnce(view(true)).mockResolvedValue(view());
    endpoints.restartEngine.mockResolvedValue({ status: 'restarting' });
    endpoints.aiStatus
      .mockResolvedValueOnce({ ...status, endpoint: 'http://old-gateway:8081' })
      .mockResolvedValue({ ...status, endpoint: 'http://new-gateway:8080' });
    endpoints.aiModels.mockResolvedValueOnce({ models: ['old-model'] }).mockResolvedValue({ models: ['new-model'] });
    await open('ai');
    expect(screen.getByText('http://old-gateway:8081')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Restart engine' }));
    await settle();

    expect(screen.getByText('http://new-gateway:8080')).toBeTruthy();
    expect(screen.queryByText('http://old-gateway:8081')).toBeNull();
    const models = [...screen.getByLabelText<HTMLSelectElement>('Default model').options].map((option) => option.value);
    expect(models).toContain('new-model');
    expect(models).not.toContain('old-model');
  });

  it('opens Advanced while it holds an unsaved change', async () => {
    await open('ai');

    await fireEvent.input(screen.getByLabelText('Timeout (seconds)'), { target: { value: '30' } });

    expect(screen.getByText('Advanced (1)').closest('details')?.open).toBe(true);
  });

  it('shows the engine validation error', async () => {
    endpoints.updateSettings.mockRejectedValue(
      new Error('validation failed: AI_TIMEOUT_SECONDS must be a positive integer'),
    );
    await open('ai');

    await fireEvent.input(screen.getByLabelText('Timeout (seconds)'), { target: { value: '30' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }));
    await settle();

    expect(screen.getByRole('alert').textContent).toMatch(/AI_TIMEOUT_SECONDS/);
  });

  it('blocks Connect Google account and the restart while changes are unsaved', async () => {
    await open('sheets');

    const connect = screen.getByRole<HTMLButtonElement>('button', { name: 'Connect Google account' });
    expect(connect.disabled).toBe(false);
    await fireEvent.input(screen.getByLabelText('Spreadsheet ID'), { target: { value: 'sheet-1' } });
    expect(connect.disabled).toBe(true);
  });

  it('returns from a Google sign-in to its tab and drops the outcome from the address', async () => {
    appState.page.url = new URL('http://127.0.0.1:5173/ui/settings?google=connected&purpose=sheets');
    render(SettingsPage);
    await settle();

    expect(toastMock.push).toHaveBeenCalledWith('success', 'Google account connected. Restart the engine to use it.');
    expect(navigation.goto).toHaveBeenCalledWith(new URL('http://127.0.0.1:5173/ui/settings?tab=sheets'), {
      replaceState: true,
      noScroll: true,
      keepFocus: true,
    });
    expect(screen.getByLabelText('Spreadsheet ID')).toBeTruthy();
  });

  it('leaves the address alone on a normal visit', async () => {
    await open('market');

    expect(navigation.goto).not.toHaveBeenCalled();
  });

  it('Given the engine settings are still loading When the page opens Then the tabs show at once with a skeleton', async () => {
    endpoints.settings.mockReturnValue(new Promise(() => undefined));
    await open('market');

    expect(screen.getByRole('navigation', { name: 'Settings sections' })).toBeTruthy();
    expect(screen.getByText('Loading the engine settings…')).toBeTruthy();
  });

  it('Given the engine settings fail When the page opens Then the browser preferences stay usable and Retry loads again', async () => {
    endpoints.settings.mockRejectedValueOnce(new TypeError('fetch failed'));
    await open();

    expect(screen.getByText('This browser')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'API token' })).toBeTruthy();
    expect(screen.getByText('Could not load the engine settings')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await settle();

    expect(endpoints.settings).toHaveBeenCalledTimes(2);
    expect(screen.getByLabelText('Log level')).toBeTruthy();
  });

  it('Given unsaved changes When saving Then Save is busy until the engine answers', async () => {
    endpoints.updateSettings.mockReturnValue(new Promise(() => undefined));
    await open('market');

    await fireEvent.input(screen.getByLabelText('Market data server URL'), { target: { value: 'http://md:8000' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }));
    await settle();

    expect(screen.getByRole('button', { name: 'Saving…' }).getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Discard' }).disabled).toBe(true);
  });

  it('Given the AI tab When the status is still loading Then it shows a skeleton, not a blank card', async () => {
    endpoints.aiStatus.mockReturnValue(new Promise(() => undefined));
    await open('ai');

    expect(screen.getByText('Loading the AI status…')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeTruthy();
  });

  it('Given the goal pool fails to load When the Goals tab opens Then it shows the error with a Retry', async () => {
    endpoints.preferences.mockRejectedValueOnce(new Error('Preferences tab missing'));
    await open('goals');

    expect(screen.getByText('Could not load the goal settings')).toBeTruthy();
    expect(screen.getByText('Preferences tab missing')).toBeTruthy();
  });
});
