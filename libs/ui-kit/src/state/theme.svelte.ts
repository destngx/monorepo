export type Theme = 'system' | 'light' | 'dark';

export const THEMES: readonly Theme[] = ['system', 'light', 'dark'];

const DARK_QUERY = '(prefers-color-scheme: dark)';

function systemIsDark(): boolean {
  return globalThis.matchMedia?.(DARK_QUERY).matches ?? false;
}

/** Applies a theme to the page: the .dark class on <html> and the matching color-scheme. */
export function applyTheme(theme: Theme, prefersDark = systemIsDark()): void {
  const isDark = theme === 'dark' || (theme === 'system' && prefersDark);
  document.documentElement.classList.toggle('dark', isDark);
  document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
}

/**
 * The theme setting, saved in localStorage under `key` (e.g. "hw.theme").
 * "system" is the default and is stored as no key at all, so the first-paint
 * script in app.html and this class read the same thing.
 */
export class ThemeSetting {
  readonly key: string;
  value = $state<Theme>('system');

  constructor(key: string) {
    this.key = key;
    this.value = this.#read();
  }

  set(theme: Theme): void {
    this.value = theme;
    try {
      if (theme === 'system') localStorage.removeItem(this.key);
      else localStorage.setItem(this.key, theme);
    } catch {
      // Storage unavailable: the theme applies to this page only.
    }
    applyTheme(theme);
  }

  /** The next theme in the order system, light, dark. */
  cycle(): void {
    this.set(THEMES[(THEMES.indexOf(this.value) + 1) % THEMES.length]);
  }

  /** Follows later system changes while the setting is "system". Returns the function that stops it. */
  watchSystem(): () => void {
    const query = globalThis.matchMedia?.(DARK_QUERY);
    if (!query) return () => undefined;
    const change = (event: MediaQueryListEvent) => {
      if (this.value === 'system') applyTheme('system', event.matches);
    };
    query.addEventListener('change', change);
    return () => query.removeEventListener('change', change);
  }

  #read(): Theme {
    try {
      const saved = localStorage.getItem(this.key);
      return saved === 'light' || saved === 'dark' ? saved : 'system';
    } catch {
      return 'system';
    }
  }
}

export function createTheme(key: string): ThemeSetting {
  return new ThemeSetting(key);
}

/**
 * The inline script for app.html that applies the saved theme before the
 * first paint, so dark mode never flashes light. app.html cannot import it:
 * paste the output into a <script> in its <head>.
 */
export function themeScript(key: string): string {
  return `try{var t=localStorage.getItem(${JSON.stringify(key)}),d=t==='dark'||(t!=='light'&&!!window.matchMedia&&matchMedia('${DARK_QUERY}').matches);document.documentElement.classList.toggle('dark',d);document.documentElement.style.colorScheme=d?'dark':'light'}catch(e){}`;
}
