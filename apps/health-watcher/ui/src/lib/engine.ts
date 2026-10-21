import type { EngineSettings, EngineSettingsInput, EngineSettingsView, RestartField } from './api/types';

/** The Engine settings form: the saved settings plus the key field, which starts empty. */
export interface EngineForm extends EngineSettings {
  aiApiKey: string;
  shouldClearKey: boolean;
}

export function engineForm(view: EngineSettingsView): EngineForm {
  return { ...view.settings, aiApiKey: '', shouldClearKey: false };
}

/** The save request. An empty key field keeps the saved key unless removal is ticked. */
export function settingsInput(form: EngineForm): EngineSettingsInput {
  const { aiApiKey, shouldClearKey, ...settings } = form;
  const key = aiApiKey.trim();
  if (shouldClearKey) return { ...settings, aiApiKey: '' };
  return key ? { ...settings, aiApiKey: key } : settings;
}

export function isDirty(form: EngineForm, view: EngineSettingsView): boolean {
  if (form.shouldClearKey || form.aiApiKey.trim()) return true;
  const saved = view.settings;
  return (Object.keys(saved) as (keyof EngineSettings)[]).some((k) => form[k] !== saved[k]);
}

const restartLabels: Record<RestartField, string> = { host: 'host', port: 'port', maxBatchMb: 'largest upload' };

export function restartNote(fields: RestartField[]): string {
  return fields.length ? `Restart the engine to apply: ${fields.map((f) => restartLabels[f]).join(', ')}.` : '';
}

const fieldLabels: Record<keyof EngineSettings | 'aiApiKey', string> = {
  host: 'host',
  port: 'port',
  maxFileMb: 'largest file',
  maxBatchMb: 'largest upload',
  macOcrBin: 'mac-ocr program',
  workers: 'background jobs',
  aiBaseUrl: 'AI address',
  aiApiKey: 'API key',
  aiTimeoutSeconds: 'AI time limit',
};

export function rejectedNote(fields: EngineSettingsView['rejected']): string {
  return fields.length
    ? `These saved values were not valid and use their defaults until you save: ${fields.map((f) => fieldLabels[f]).join(', ')}.`
    : '';
}
