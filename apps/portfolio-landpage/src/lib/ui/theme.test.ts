import { beforeEach, describe, expect, it } from 'vitest';
import { createThemeController } from './theme.svelte';

describe('theme controller', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.dataset.theme = 'light';
  });

  it('reads the theme already applied by the inline boot script', () => {
    document.documentElement.dataset.theme = 'dark';

    expect(createThemeController().current).toBe('dark');
  });

  it('toggles, persists, and applies the theme to <html>', () => {
    const theme = createThemeController();

    theme.toggle();

    expect(theme.current).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(localStorage.getItem('theme')).toBe('dark');

    theme.toggle();

    expect(theme.current).toBe('light');
    expect(localStorage.getItem('theme')).toBe('light');
  });
});
