import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChatEvent } from '$lib/api/types';
import { MASK_TOKEN_LABEL } from '$lib/components/maskTokens';
import { mask } from '$lib/state/mask.svelte';

const endpoints = vi.hoisted(() => ({ chat: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
const navigation = vi.hoisted(() => ({ goto: vi.fn() }));
vi.mock('$app/navigation', () => navigation);

import ChatPanel from './ChatPanel.svelte';
import { chat } from './chat.svelte';
import { pendingFills } from '$lib/fill/pending.svelte';

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
};

/** A reply that sends `events`, then waits until stopped. */
function reply(events: ChatEvent[]) {
  return (_messages: unknown, _page: unknown, signal: AbortSignal) =>
    (async function* () {
      for (const event of events) yield event;
      await new Promise((_, reject) => signal.addEventListener('abort', () => reject(new Error('aborted'))));
    })();
}

const page = { path: '/chat', title: 'Assistant' };

async function ask(text: string) {
  await fireEvent.input(screen.getByLabelText('Message'), { target: { value: text } });
  await fireEvent.click(screen.getByRole('button', { name: 'Send' }));
  await flush();
}

describe('ChatPanel', () => {
  beforeEach(() => {
    chat.clear();
    chat.draft = '';
    mask.hide();
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
    if (!('scrollTo' in Element.prototype)) Object.assign(Element.prototype, { scrollTo: () => undefined });
  });

  it('Given a question was sent When the reply has not started Then it shows typing, a busy Send and a Stop', async () => {
    endpoints.chat.mockImplementation(reply([]));
    render(ChatPanel, { page });

    await ask('How is my budget?');

    expect(screen.getByRole('status', { name: 'Assistant is typing' })).toBeTruthy();
    const send = screen.getByRole('button', { name: 'Send' });
    expect(send.getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole('button', { name: 'Stop' })).toBeTruthy();
    chat.stop();
    await flush();
  });

  it('Given the reply is streaming When text arrives Then it says it is still answering', async () => {
    endpoints.chat.mockImplementation(reply([{ type: 'text', delta: 'Your budget is on track.' }]));
    render(ChatPanel, { page });

    await ask('How is my budget?');

    expect(screen.getByText('Your budget is on track.')).toBeTruthy();
    expect(screen.getByText('Answering…')).toBeTruthy();
    chat.stop();
    await flush();
  });

  it('Given a reply is stopped When nothing was written Then it says so with a Retry, and Send is ready again', async () => {
    endpoints.chat.mockImplementation(reply([]));
    render(ChatPanel, { page });
    await ask('How is my budget?');

    await fireEvent.click(screen.getByRole('button', { name: 'Stop' }));
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Stopped.');
    expect(screen.getByRole('button', { name: 'Retry' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Send' }).getAttribute('aria-busy')).toBeNull();
  });

  it('Given a question that quotes an amount When masked Then the amount in the bubble is a mask token', async () => {
    endpoints.chat.mockImplementation(reply([{ type: 'text', delta: 'You spent 2,000,000 ₫ on food.' }]));
    render(ChatPanel, { page });

    await ask('Can I spend 5,000,000 VND this month?');

    expect(screen.queryByText(/5,000,000/)).toBeNull();
    expect(screen.queryByText(/2,000,000/)).toBeNull();
    expect(screen.getAllByRole('button', { name: MASK_TOKEN_LABEL }).length).toBeGreaterThanOrEqual(1);
    chat.stop();
    await flush();
  });

  it('Given a conversation When Clear conversation is pressed Then it asks before removing the messages', async () => {
    endpoints.chat.mockImplementation(reply([{ type: 'text', delta: 'On track.' }]));
    render(ChatPanel, { page });
    await ask('How is my budget?');
    chat.stop();
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Clear conversation' }));
    expect(chat.entries.length).toBeGreaterThan(0);

    await fireEvent.click(screen.getByRole('button', { name: 'Clear', hidden: true }));
    expect(chat.entries).toEqual([]);
  });

  it('Given a message being typed When the panel closes and opens again Then the text is still there', async () => {
    const first = render(ChatPanel, { page });
    await fireEvent.input(screen.getByLabelText('Message'), { target: { value: 'Half a question' } });
    first.unmount();

    render(ChatPanel, { page });
    expect(screen.getByLabelText<HTMLTextAreaElement>('Message').value).toBe('Half a question');
  });
});

describe('ChatPanel: a filled form (WM-FILL-7)', () => {
  beforeEach(() => {
    chat.clear();
    chat.draft = '';
  });

  it('offers to open the form the chat filled and hands the fill to that page', async () => {
    const form = {
      form: 'goal' as const,
      title: 'Goal',
      page: '/accounts/goals',
      text: 'save 50M for a trip by June 2027',
      result: {
        form: 'goal' as const,
        values: { name: 'Trip', target: 50_000_000 },
        filled: ['name', 'target'],
        notes: [],
      },
    };
    endpoints.chat.mockImplementation(
      reply([
        { type: 'tool', tool: 'fillForm', status: 'running' },
        { type: 'tool', tool: 'fillForm', status: 'ok' },
        { type: 'form', form },
        { type: 'text', delta: 'Open the form to check it.' },
        { type: 'done' },
      ]),
    );
    render(ChatPanel, { page });

    await ask('Add a goal: 50M for a trip by June 2027');
    expect(screen.getByText('Filling the form')).toBeTruthy();
    expect(screen.getByText('Filled 2 fields')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Open the form' }));
    expect(navigation.goto).toHaveBeenCalledWith('/ui/accounts/goals');
    expect(pendingFills.take('goal')).toEqual(form);
    chat.stop();
  });
});
