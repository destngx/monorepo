import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import PageHeadline from './PageHeadline.svelte';

describe('PageHeadline', () => {
  it('asks the question and answers it with the status icon and label, never colour alone', () => {
    const { container } = render(PageHeadline, { question: 'Am I okay?', tone: 'warning', title: 'Watch spending' });

    const section = container.querySelector('section') as HTMLElement;
    expect(section.getAttribute('data-status')).toBe('warning');
    expect(section.className).toContain('border-l-amber-500');
    expect(section.getAttribute('aria-labelledby')).toBe('page-headline');
    expect(screen.getByText('Am I okay?')).toBeTruthy();
    const heading = screen.getByRole('heading', { level: 2 });
    expect(heading.id).toBe('page-headline');
    expect(heading.querySelector('svg')).toBeTruthy();
    expect(heading.textContent).toContain('Warning:');
    expect(heading.textContent).toContain('Watch spending');
  });
});
