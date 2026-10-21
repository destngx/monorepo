import { api } from '$lib/api/endpoints';
import type { ChatPageContext } from '$lib/api/types';
import {
  applyEvent,
  editEntry,
  HISTORY_KEY,
  MAX_STORED,
  parseHistory,
  toMessages,
  retryFrom,
  type ChatEntry,
} from './history';

function load(): ChatEntry[] {
  try {
    return parseHistory(globalThis.localStorage?.getItem(HISTORY_KEY) ?? null);
  } catch {
    return [];
  }
}

function save(entries: ChatEntry[]): void {
  try {
    if (entries.length === 0) globalThis.localStorage?.removeItem(HISTORY_KEY);
    else globalThis.localStorage?.setItem(HISTORY_KEY, JSON.stringify(entries.slice(-MAX_STORED)));
  } catch {
    // Storage can be unavailable (private mode); history then lasts for the session.
  }
}

/** The one conversation shared by the widget and the /chat page (WM-AI-11). */
class ChatStore {
  entries = $state<ChatEntry[]>(load());
  streaming = $state(false);
  /** The message being typed: it lives here so closing the widget or opening /chat keeps it. */
  draft = $state('');
  #controller: AbortController | null = null;

  async send(text: string, page?: ChatPageContext): Promise<void> {
    const content = text.trim();
    if (!content || this.streaming) return;
    this.entries.push({ role: 'user', content });
    await this.#reply(page);
  }

  /**
   * Answers the message at index again: a user message, or the user message an
   * assistant reply answers. Everything after it is dropped.
   */
  async retry(index: number, page?: ChatPageContext): Promise<void> {
    if (this.streaming) return;
    const entries = retryFrom(this.entries, index);
    if (!entries) return;
    this.entries = entries;
    await this.#reply(page);
  }

  /** Rewrites one message in place, keeping the rest of the conversation. */
  edit(index: number, content: string): boolean {
    if (this.streaming) return false;
    const entries = editEntry(this.entries, index, content);
    if (!entries) return false;
    this.entries = entries;
    save(this.entries);
    return true;
  }

  /** Rewrites a user message and answers it again, dropping what followed. */
  async editAndRetry(index: number, content: string, page?: ChatPageContext): Promise<void> {
    if (this.entries[index]?.role !== 'user' || !this.edit(index, content)) return;
    await this.retry(index, page);
  }

  async #reply(page?: ChatPageContext): Promise<void> {
    // Snapshot before the placeholder so the newest user message is included.
    const messages = toMessages(this.entries);
    this.entries.push({ role: 'assistant', content: '' });
    const index = this.entries.length - 1;
    const controller = new AbortController();
    this.#controller = controller;
    this.streaming = true;
    try {
      for await (const event of api.chat(messages, page, controller.signal)) {
        this.entries[index] = applyEvent(this.entries[index], event);
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        const message = error instanceof Error ? error.message : String(error);
        this.entries[index] = applyEvent(this.entries[index], { type: 'error', message });
      }
    } finally {
      if (controller.signal.aborted && !this.entries[index].content) {
        this.entries[index] = { ...this.entries[index], error: 'Stopped.' };
      }
      this.streaming = false;
      this.#controller = null;
      save(this.entries);
    }
  }

  stop(): void {
    this.#controller?.abort();
  }

  clear(): void {
    this.stop();
    this.entries = [];
    save(this.entries);
  }
}

export const chat = new ChatStore();
