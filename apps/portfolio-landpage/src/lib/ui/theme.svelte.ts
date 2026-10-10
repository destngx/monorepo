export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'theme';

const readTheme = (): Theme => (document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light');

/**
 * Client-side theme state. The initial value is applied before hydration by the
 * inline script in app.html, so this only mirrors and updates it.
 */
export function createThemeController() {
  let current = $state<Theme>(readTheme());

  return {
    get current() {
      return current;
    },
    toggle() {
      current = current === 'dark' ? 'light' : 'dark';
      document.documentElement.dataset.theme = current;
      try {
        localStorage.setItem(STORAGE_KEY, current);
      } catch (error) {
        // Private mode or blocked storage: the theme still applies for this visit.
        console.warn('Could not persist theme preference', error);
      }
    },
  };
}
