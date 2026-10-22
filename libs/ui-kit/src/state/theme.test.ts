import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { applyTheme, createTheme, themeScript } from './theme.svelte';

const KEY = 'test.theme';
const isDark = () => document.documentElement.classList.contains('dark');

/** A matchMedia whose dark preference the test sets and changes. */
function fakeSystem(dark: boolean) {
  const listeners = new Set<(event: MediaQueryListEvent) => void>();
  const query = {
    get matches() {
      return dark;
    },
    addEventListener: (_: string, listener: (event: MediaQueryListEvent) => void) => listeners.add(listener),
    removeEventListener: (_: string, listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener),
  };
  vi.stubGlobal('matchMedia', () => query);
  return {
    change(next: boolean) {
      dark = next;
      for (const listener of listeners) listener({ matches: next } as MediaQueryListEvent);
    },
    listeners,
  };
}

/** Runs the script the way app.html does: the exact string, as code. */
function runInlineScript(source: string) {
  // eslint-disable-next-line @typescript-eslint/no-implied-eval -- the test runs the string app.html will run
  const script = new Function(source) as () => void;
  script();
}

describe('theme setting', () => {
  beforeEach(() => {
    localStorage.clear();
    applyTheme('light', false);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('Given nothing saved When the app starts Then the setting is system', () => {
    expect(createTheme(KEY).value).toBe('system');
  });

  it('Given dark is chosen When set Then the page turns dark and the choice is saved', () => {
    fakeSystem(false);
    const theme = createTheme(KEY);
    theme.set('dark');
    expect(isDark()).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');
    expect(localStorage.getItem(KEY)).toBe('dark');
    expect(createTheme(KEY).value).toBe('dark');
  });

  it('Given system is chosen When set Then nothing is stored and the system preference applies', () => {
    fakeSystem(true);
    const theme = createTheme(KEY);
    theme.set('light');
    theme.set('system');
    expect(localStorage.getItem(KEY)).toBeNull();
    expect(isDark()).toBe(true);
  });

  it('Given the toggle When cycled Then it goes system, light, dark and back', () => {
    fakeSystem(false);
    const theme = createTheme(KEY);
    const seen = [theme.value];
    for (let i = 0; i < 3; i++) {
      theme.cycle();
      seen.push(theme.value);
    }
    expect(seen).toEqual(['system', 'light', 'dark', 'system']);
  });

  it('Given the system setting When the system switches to dark Then the page follows; a fixed choice does not', () => {
    const system = fakeSystem(false);
    const theme = createTheme(KEY);
    const stop = theme.watchSystem();

    system.change(true);
    expect(isDark()).toBe(true);

    theme.set('light');
    system.change(true);
    expect(isDark()).toBe(false);

    stop();
    expect(system.listeners.size).toBe(0);
  });

  it('Given the first-paint script When it runs with a saved dark theme Then the page starts dark', () => {
    fakeSystem(false);
    localStorage.setItem(KEY, 'dark');
    runInlineScript(themeScript(KEY));
    expect(isDark()).toBe(true);
  });

  it('Given the first-paint script When nothing is saved Then it follows the system', () => {
    fakeSystem(true);
    runInlineScript(themeScript(KEY));
    expect(isDark()).toBe(true);
  });
});
