import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { runInThisContext } from 'node:vm';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { endBootScreen } from '$lib/boot';

// The shell the browser shows before the app's JS and first data load.
const shell = readFileSync(resolve(__dirname, 'app.html'), 'utf8')
  .replace('%sveltekit.head%', '')
  .replace('%sveltekit.body%', '<script>/* app start */</script>');
const parse = () => new DOMParser().parseFromString(shell, 'text/html');

/** Runs app.html's head script against this window, as the browser does before the first paint. */
function applySavedTheme({ theme, prefersDark }: { theme?: string; prefersDark: boolean }) {
  vi.stubGlobal('localStorage', { getItem: (key: string) => (key === 'wm.theme' ? (theme ?? null) : null) });
  vi.stubGlobal('matchMedia', (query: string) => ({ matches: prefersDark && query.includes('dark') }));
  runInThisContext(parse().head.querySelector('script')?.textContent ?? '');
}

describe('app shell', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    document.documentElement.className = '';
    document.documentElement.style.colorScheme = '';
  });

  it('shows a loading indicator until the app renders', () => {
    const page = parse();
    expect(page.querySelector('[role="status"]')?.textContent).toMatch(/Loading/);

    endBootScreen(page);

    expect(page.querySelector('[role="status"]')).toBeNull();
    expect(page.body.querySelector('script')).toBeTruthy();
  });

  it.each([
    { theme: 'dark', prefersDark: false, dark: true },
    { theme: 'light', prefersDark: true, dark: false },
    { theme: undefined, prefersDark: true, dark: true },
    { theme: 'system', prefersDark: false, dark: false },
  ])('applies the saved theme before the first paint: $theme, system dark $prefersDark', ({ dark, ...saved }) => {
    applySavedTheme(saved);

    expect(document.documentElement.classList.contains('dark')).toBe(dark);
    expect(document.documentElement.style.colorScheme).toBe(dark ? 'dark' : 'light');
  });
});
