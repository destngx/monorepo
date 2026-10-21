import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import FilmGuide from './FilmGuide.svelte';

describe('FilmGuide', () => {
  it('is folded away until opened, and asks for the written report too', () => {
    const { container } = render(FilmGuide);
    expect(container.querySelector('details')?.open).toBe(false);
    expect(screen.getByText('How to photograph an X-ray or CT film')).toBeTruthy();
    expect(screen.getByText(/Upload the written radiology report too/)).toBeTruthy();
  });
  it('can start open', () => {
    const { container } = render(FilmGuide, { open: true });
    expect(container.querySelector('details')?.open).toBe(true);
  });
});
