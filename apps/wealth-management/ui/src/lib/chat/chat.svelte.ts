import { api } from '$lib/api/endpoints';
import type { ChatPageContext } from '$lib/api/types';
import {
  applyEvent,
  HISTORY_KEY,
  MAX_STORED,
  parseHistory,
  toMessages,
  withoutLastReply,
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
  #controller: AbortController | null = null;

  async send(text: string, page?: ChatPageContext): Promise<void> {
    const content = text.trim();
    if (!content || this.streaming) return;
    this.entries.push({ role: 'user', content });
    await this.#reply(page);
  }

  /** Answers the last user message again, replacing the reply it got. */
  async retry(page?: ChatPageContext): Promise<void> {
    if (this.streaming) return;
    const entries = withoutLastReply(this.entries);
    if (!entries) return;
    this.entries = entries;
    await this.#reply(page);
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
