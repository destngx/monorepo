import { render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

vi.mock('$lib/api/endpoints', () => ({ api: {} }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import ChatPage from './+page.svelte';

// jsdom has no element scrolling; the panel keeps the newest message in view.
if (!('scrollTo' in Element.prototype)) Object.assign(Element.prototype, { scrollTo: () => undefined });

describe('Assistant page', () => {
  it('Given the assistant page When it opens Then the conversation and the message box show without any request', () => {
    render(ChatPage);

    expect(screen.getByRole('heading', { level: 1, name: 'Assistant' })).toBeTruthy();
    expect(screen.getByLabelText('Message')).toBeTruthy();
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Send' }).disabled).toBe(true);
  });
});
