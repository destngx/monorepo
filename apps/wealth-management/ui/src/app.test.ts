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
function applySavedTheme({
  theme,
  prefersDark,
  sidebarCollapsed,
}: {
  theme?: string;
  prefersDark: boolean;
  sidebarCollapsed?: string;
}) {
  const saved: Record<string, string | undefined> = { 'wm.theme': theme, 'wm.sidebarCollapsed': sidebarCollapsed };
  vi.stubGlobal('localStorage', { getItem: (key: string) => saved[key] ?? null });
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
    expect(page.querySelector('[role="status"] > .boot-sr')?.textContent).toBe('Loading Wealth…');

    endBootScreen(page);

    expect(page.querySelector('[role="status"]')).toBeNull();
    expect(page.body.querySelector('script')).toBeTruthy();
  });

  it('outlines the app shell while loading, so the first render lands in place', () => {
    const boot = parse().getElementById('boot');

    expect(boot?.getAttribute('aria-busy')).toBe('true');
    for (const part of ['bar', 'side', 'header', 'content', 'tabs']) {
      const el = boot?.querySelector(`.boot-${part}`);
      expect(el, part).toBeTruthy();
      expect(el?.closest('[aria-hidden="true"]'), part).toBeTruthy();
    }
  });

  it.each([
    { saved: 'true', collapsed: true },
    { saved: undefined, collapsed: false },
  ])('outlines the sidebar at its saved width: collapsed $saved', ({ saved, collapsed }) => {
    applySavedTheme({ prefersDark: false, sidebarCollapsed: saved });

    expect(document.documentElement.dataset.bootSidebar).toBe(collapsed ? 'collapsed' : undefined);

    endBootScreen(document);
    expect(document.documentElement.dataset.bootSidebar).toBeUndefined();
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
