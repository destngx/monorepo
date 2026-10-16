import type { EngineSetting, SettingsChanges } from '$lib/api/types';

/** The tabs of the Settings page; engine groups render in the tab that lists them. */
export const sections = [
  { id: 'general', label: 'General', groups: ['general'] },
  { id: 'connections', label: 'Connections', groups: [] },
  { id: 'sheets', label: 'Google Sheets', groups: ['sheets'] },
  { id: 'ai', label: 'AI', groups: ['ai', 'search'] },
  { id: 'market', label: 'Market data', groups: ['market'] },
  { id: 'cache', label: 'Cache', groups: ['cache'] },
  { id: 'capture', label: 'Smart Capture', groups: ['capture'] },
  // Stored in the sheet's Preferences tab, not in the engine settings (WM-SET-8).
  { id: 'goals', label: 'Goal plan', groups: [] },
] as const satisfies readonly { id: string; label: string; groups: readonly string[] }[];

export type SectionId = (typeof sections)[number]['id'];

const isSection = (id: string | null): id is SectionId => sections.some((section) => section.id === id);

/**
 * The open tab: `?tab=`, else the tab a Google sign-in started from (the
 * engine's callback returns with `purpose`), else General.
 */
export function sectionFromUrl(url: URL): SectionId {
  const tab = url.searchParams.get('tab');
  if (isSection(tab)) return tab;
  if (url.searchParams.get('purpose') === 'sheets') return 'sheets';
  return 'general';
}

/** Settings in engine order, with the rarely changed ones apart. */
export function splitAdvanced(settings: EngineSetting[]): { main: EngineSetting[]; advanced: EngineSetting[] } {
  return {
    main: settings.filter((setting) => !setting.advanced),
    advanced: settings.filter((setting) => setting.advanced),
  };
}

/** The tabs holding at least one unsaved change. */
export function sectionsWithChanges(settings: EngineSetting[], changes: SettingsChanges): Set<SectionId> {
  const changed = new Set<SectionId>();
  for (const setting of settings) {
    if (!(setting.key in changes)) continue;
    const section = sections.find((candidate) => (candidate.groups as readonly string[]).includes(setting.group));
    if (section) changed.add(section.id);
  }
  return changed;
}

/** Options for a select; a stored value the list no longer offers stays selectable. */
export function choiceOptions(options: readonly string[], current: string): string[] {
  return current && !options.includes(current) ? [current, ...options] : [...options];
}
