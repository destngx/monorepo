import { describe, expect, it } from 'vitest';
import type { EngineSettingsView } from './api/types';
import { engineForm, isDirty, rejectedNote, restartNote, settingsInput } from './engine';

function view(over: Partial<EngineSettingsView> = {}): EngineSettingsView {
  return {
    settings: {
      host: '127.0.0.1',
      port: 9191,
      maxFileMb: 50,
      maxBatchMb: 200,
      macOcrBin: '',
      workers: 2,
      aiBaseUrl: 'http://localhost:8080/v1',
      aiTimeoutSeconds: 180,
    },
    aiApiKeySet: true,
    aiModel: 'gpt-6-luna',
    aiEffort: 'low',
    defaultMacOcrBin: '/opt/mac-ocr',
    macOcrFound: true,
    listening: '127.0.0.1:9191',
    restartFields: [],
    rejected: [],
    stamp: { updatedAt: null, updatedBy: '', updatedByName: '' },
    ...over,
  };
}

describe('engine settings form', () => {
  it('given saved settings when opening the form then the key field starts empty and nothing is dirty', () => {
    const form = engineForm(view());
    expect(form.aiApiKey).toBe('');
    expect(form.shouldClearKey).toBe(false);
    expect(isDirty(form, view())).toBe(false);
  });

  it('given an untouched key field when saving then the saved key is kept', () => {
    const input = settingsInput(engineForm(view()));
    expect('aiApiKey' in input).toBe(false);
    expect(input.workers).toBe(2);
  });

  it('given a new key or a removal when saving then it is sent', () => {
    const form = engineForm(view());
    form.aiApiKey = '  sk-new  ';
    expect(settingsInput(form).aiApiKey).toBe('sk-new');
    expect(isDirty(form, view())).toBe(true);
    form.aiApiKey = '';
    form.shouldClearKey = true;
    expect(settingsInput(form).aiApiKey).toBe('');
    expect(isDirty(form, view())).toBe(true);
  });

  it('given a changed field when comparing then the form is dirty', () => {
    const form = engineForm(view());
    form.port = 9292;
    expect(isDirty(form, view())).toBe(true);
  });

  it('given fields waiting for a restart when describing them then they are named', () => {
    expect(restartNote([])).toBe('');
    expect(restartNote(['host', 'maxBatchMb'])).toBe('Restart the engine to apply: host, largest upload.');
  });

  it('given stored values that failed the checks when describing them then they are named', () => {
    expect(rejectedNote([])).toBe('');
    expect(rejectedNote(['port', 'aiApiKey'])).toBe(
      'These saved values were not valid and use their defaults until you save: port, API key.',
    );
  });
});
