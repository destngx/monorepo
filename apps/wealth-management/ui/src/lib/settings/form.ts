import type { EngineSetting, EngineSettings, SettingGroup, SettingsChanges } from '$lib/api/types';

/**
 * The editor's state for one setting. Plain settings start at the stored
 * value; secrets start empty (their value is never sent to the browser) and
 * an empty secret draft means "keep the stored value".
 */
export interface SettingDraft {
  value: string;
  clear: boolean;
}

export type Drafts = Record<string, SettingDraft>;

export function initialDrafts(settings: EngineSetting[]): Drafts {
  return Object.fromEntries(
    settings.map((setting) => [setting.key, { value: setting.secret ? '' : (setting.value ?? ''), clear: false }]),
  );
}

/** The changes to send: only settings whose draft differs from the stored value. */
export function pendingChanges(settings: EngineSetting[], drafts: Drafts): SettingsChanges {
  const changes: SettingsChanges = {};
  for (const setting of settings) {
    const draft = drafts[setting.key];
    if (!draft) continue;
    const value = draft.value.trim();
    if (setting.secret) {
      if (draft.clear) {
        if (setting.isSet) changes[setting.key] = null;
      } else if (value) changes[setting.key] = value;
    } else if (value !== (setting.value ?? '')) {
      changes[setting.key] = value || null;
    }
  }
  return changes;
}

export interface GroupedSettings {
  group: SettingGroup;
  settings: EngineSetting[];
}

/** Settings by group, in the engine's order, without empty groups. */
export function groupSettings(view: EngineSettings): GroupedSettings[] {
  return view.groups
    .map((group) => ({ group, settings: view.settings.filter((setting) => setting.group === group.id) }))
    .filter(({ settings }) => settings.length > 0);
}

export interface WaitOptions {
  timeoutMs?: number;
  intervalMs?: number;
  sleep?: (ms: number) => Promise<void>;
  now?: () => number;
}

/**
 * Polls load until the restarted engine answers: a response with
 * restartRequired still true comes from the old process, which may answer
 * before it shuts down. Throws after timeoutMs.
 */
export async function waitForRestart(
  load: () => Promise<EngineSettings>,
  {
    timeoutMs = 60_000,
    intervalMs = 1_000,
    sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
    now = Date.now,
  }: WaitOptions = {},
): Promise<EngineSettings> {
  const deadline = now() + timeoutMs;
  let last: unknown;
  while (now() < deadline) {
    await sleep(intervalMs);
    try {
      const view = await load();
      if (!view.restartRequired) return view;
    } catch (failure) {
      last = failure;
    }
  }
  const reason = last instanceof Error ? `: ${last.message}` : '';
  throw new Error(`The engine did not come back within ${Math.round(timeoutMs / 1000)} s${reason}. Check its log.`);
}
