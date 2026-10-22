import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Aurora from './Aurora.svelte';

describe('Aurora background (WM-UI-5)', () => {
  it('Given the app shell, when rendered, then the background is decoration hidden from screen readers and pointers', () => {
    const { container } = render(Aurora);

    const layer = container.querySelector('[data-aurora]');
    expect(layer?.getAttribute('aria-hidden')).toBe('true');
    expect(layer?.className).toContain('pointer-events-none');
    expect(layer?.className).toContain('fixed');
    expect(layer?.querySelectorAll('.blob')).toHaveLength(4);
    expect(layer?.querySelector('.halftone')).toBeTruthy();
  });
});
