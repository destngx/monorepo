import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/api/endpoints', () => ({ api: { chat: vi.fn() } }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import ChatWidget from './ChatWidget.svelte';

const page = { path: '/budget', title: 'Budget' };

describe('ChatWidget', () => {
  beforeEach(() => {
    if (!('scrollTo' in Element.prototype)) Object.assign(Element.prototype, { scrollTo: () => undefined });
  });

  it('Given the assistant is closed When it opens Then the message box has focus', async () => {
    render(ChatWidget, { page });
    await fireEvent.click(screen.getByRole('button', { name: 'Ask the assistant' }));
    await tick();
    expect(document.activeElement).toBe(screen.getByLabelText('Message'));
  });

  it('Given the assistant is open When Escape is pressed Then it closes and focus returns to its button', async () => {
    render(ChatWidget, { page });
    await fireEvent.click(screen.getByRole('button', { name: 'Ask the assistant' }));
    await tick();

    await fireEvent.keyDown(screen.getByLabelText('Message'), { key: 'Escape' });
    expect(screen.queryByRole('dialog', { name: 'Assistant' })).toBeNull();
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Ask the assistant' }));
  });
});
