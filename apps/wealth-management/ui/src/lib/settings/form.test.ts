import type { EngineSetting, EngineSettings } from '$lib/api/types';
import { describe, expect, it } from 'vitest';
import { groupSettings, initialDrafts, pendingChanges, waitForRestart } from './form';

const setting = (key: string, extra: Partial<EngineSetting> = {}): EngineSetting => ({
  key,
  group: 'sheets',
  label: key,
  description: '',
  kind: 'text',
  secret: false,
  isSet: false,
  ...extra,
});

const settings = [
  setting('GOOGLE_SHEETS_ID', { isSet: true, value: 'sheet' }),
  setting('BUDGET_SHEET_TEMPLATE'),
  setting('AI_API_KEY', { group: 'ai', kind: 'secret', secret: true, isSet: true }),
  setting('EXA_API_KEY', { group: 'search', kind: 'secret', secret: true }),
];

const view = (restartRequired: boolean): EngineSettings => ({
  groups: [
    { id: 'sheets', label: 'Google Sheets', description: '' },
    { id: 'cache', label: 'Cache', description: '' },
    { id: 'ai', label: 'AI', description: '' },
  ],
  settings,
  path: '/settings.db',
  restartRequired,
});

describe('initialDrafts', () => {
  it('starts plain settings at their value and secrets empty', () => {
    const drafts = initialDrafts(settings);
    expect(drafts.GOOGLE_SHEETS_ID).toEqual({ value: 'sheet', clear: false });
    expect(drafts.AI_API_KEY).toEqual({ value: '', clear: false });
  });
});

describe('pendingChanges', () => {
  it('is empty when nothing changed', () => {
    expect(pendingChanges(settings, initialDrafts(settings))).toEqual({});
  });

  it('sends edited values trimmed and clears emptied ones', () => {
    const drafts = initialDrafts(settings);
    drafts.GOOGLE_SHEETS_ID.value = '  ';
    drafts.BUDGET_SHEET_TEMPLATE.value = ' Budget {year} ';
    expect(pendingChanges(settings, drafts)).toEqual({
      GOOGLE_SHEETS_ID: null,
      BUDGET_SHEET_TEMPLATE: 'Budget {year}',
    });
  });

  it('keeps a secret unless it is replaced or cleared', () => {
    const drafts = initialDrafts(settings);
    drafts.EXA_API_KEY.value = 'exa-key';
    expect(pendingChanges(settings, drafts)).toEqual({ EXA_API_KEY: 'exa-key' });

    drafts.AI_API_KEY.clear = true;
    drafts.EXA_API_KEY = { value: '', clear: true };
    expect(pendingChanges(settings, drafts)).toEqual({ AI_API_KEY: null });
  });
});

describe('groupSettings', () => {
  it('keeps the engine order and drops empty groups', () => {
    const groups = groupSettings(view(false)).map(({ group, settings }) => [group.id, settings.length]);
    expect(groups).toEqual([
      ['sheets', 2],
      ['ai', 1],
    ]);
  });
});

describe('waitForRestart', () => {
  const clock = () => {
    let time = 0;
    return {
      now: () => time,
      sleep: (ms: number) => {
        time += ms;
        return Promise.resolve();
      },
    };
  };

  it('waits past failures and the old process until the new engine answers', async () => {
    const answers = [Promise.resolve(view(true)), Promise.reject(new Error('down')), Promise.resolve(view(false))];
    const result = await waitForRestart(() => answers.shift()!, clock());
    expect(result.restartRequired).toBe(false);
    expect(answers).toHaveLength(0);
  });

  it('gives up after the timeout with the last error', async () => {
    await expect(
      waitForRestart(() => Promise.reject(new Error('connection refused')), { ...clock(), timeoutMs: 3000 }),
    ).rejects.toThrow(/did not come back within 3 s: connection refused/);
  });
});
