// Browser-persisted preferences. Only non-financial settings live here; the
// privacy mask is deliberately never persisted (WM-UI-2).

const TOKEN_KEY = 'wm.apiToken';
const THEME_KEY = 'wm.theme';
const SIDEBAR_KEY = 'wm.sidebarCollapsed';

export type Theme = 'system' | 'light' | 'dark';

function read(key: string): string | null {
  try {
    return globalThis.localStorage?.getItem(key) ?? null;
  } catch {
    return null;
  }
}

function write(key: string, value: string | null): void {
  try {
    if (value === null || value === '') globalThis.localStorage?.removeItem(key);
    else globalThis.localStorage?.setItem(key, value);
  } catch {
    // Storage can be unavailable (private mode); the setting then lasts for the session.
  }
}

class Persisted<T extends string> {
  #key: string;
  #value = $state<T>() as T;

  constructor(key: string, fallback: T) {
    this.#key = key;
    this.#value = (read(key) as T | null) ?? fallback;
  }

  get value(): T {
    return this.#value;
  }

  set value(next: T) {
    this.#value = next;
    write(this.#key, next);
  }
}

export const apiToken = new Persisted<string>(TOKEN_KEY, '');
export const theme = new Persisted<Theme>(THEME_KEY, 'system');
export const sidebarCollapsed = new Persisted<'true' | 'false'>(SIDEBAR_KEY, 'false');

/** Applies the theme to <html>; "system" follows prefers-color-scheme. */
export function applyTheme(selected: Theme, prefersDark: boolean): void {
  const isDark = selected === 'dark' || (selected === 'system' && prefersDark);
  document.documentElement.classList.toggle('dark', isDark);
  document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
}

// Account preselected in the add-transaction form (WM-TX-4).
export const lastUsedAccount = new Persisted<string>('wm.lastAccount', '');

// Per-use-case AI model overrides (WM-AI-4); empty uses the engine default.
export const aiModels = {
  insight: new Persisted<string>('wm.aiModel.insight', ''),
  suggest: new Persisted<string>('wm.aiModel.suggest', ''),
  vision: new Persisted<string>('wm.aiModel.vision', ''),
  chat: new Persisted<string>('wm.aiModel.chat', ''),
};

// Target allocation by asset class as JSON (WM-INV-20); parsed by $lib/investments/allocation.
export const targetAllocation = new Persisted<string>('wm.targetAllocation', '');
