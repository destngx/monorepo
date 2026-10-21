export type Theme = 'system' | 'light' | 'dark';

const KEY = 'hw.theme';

function read(): Theme {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'light' || v === 'dark' ? v : 'system';
  } catch {
    return 'system';
  }
}

/** Applies a theme to the page; app.html applies the saved one before first paint. */
export function applyTheme(theme: Theme): void {
  const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', isDark);
  document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
}

class ThemeSetting {
  value = $state<Theme>(read());

  set(theme: Theme): void {
    this.value = theme;
    try {
      if (theme === 'system') localStorage.removeItem(KEY);
      else localStorage.setItem(KEY, theme);
    } catch {
      // Storage unavailable: applies to this page only.
    }
    applyTheme(theme);
  }
}

export const theme = new ThemeSetting();
