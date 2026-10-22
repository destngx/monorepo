import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createTheme } from '../state/theme.svelte';
import { toasts } from '../state/toasts.svelte';
import Badge from './Badge.svelte';
import PageTitle from './PageTitle.svelte';
import ThemeToggle from './ThemeToggle.svelte';
import Toasts from './Toasts.svelte';

describe('Badge', () => {
  it('Given a status When shown Then it carries the status icon and label', () => {
    const { container } = render(Badge, { status: 'critical' });
    const badge = container.querySelector('[data-status="critical"]');
    expect(badge?.textContent).toBe('Critical');
    expect(badge?.querySelector('svg')).toBeTruthy();
    expect(badge?.className).toContain('tag');
  });
});

describe('PageTitle', () => {
  it('Given a site name When rendered Then the document title is "Title · Site"', () => {
    render(PageTitle, { title: 'Inbox', site: 'Health' });
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Inbox');
    expect(document.title).toBe('Inbox · Health');
  });
});

describe('ThemeToggle', () => {
  afterEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it('Given the system theme When clicked Then it moves to light and says so', async () => {
    vi.stubGlobal('matchMedia', () => ({ matches: false }));
    render(ThemeToggle, { theme: createTheme('test.theme') });
    const button = screen.getByRole('button', { name: /System theme/ });

    await fireEvent.click(button);
    expect(button.getAttribute('aria-label')).toBe('Light theme; click to change');
  });
});

describe('Toasts', () => {
  afterEach(() => {
    toasts.items = [];
  });

  it('Given an error and a success When shown Then the error is an alert, the success a status, and both dismiss', async () => {
    toasts.error(new Error('locked'), 'Could not save');
    toasts.success('Synced.');
    render(Toasts);

    expect(screen.getByRole('alert').textContent).toContain('Could not save: locked');
    expect(screen.getByRole('status').textContent).toContain('Synced.');

    for (const button of screen.getAllByRole('button', { name: 'Dismiss' })) await fireEvent.click(button);
    expect(toasts.items).toEqual([]);
  });
});
