import type { ChatEvent, ChatMessage } from '$lib/api/types';

// One chat format shared by the floating widget and the /chat page (WM-AI-11),
// so a conversation started in one continues in the other.

export interface ChatTool {
  name: string;
  status: 'running' | 'ok' | 'error';
}

export interface ChatEntry {
  role: 'user' | 'assistant';
  content: string;
  tools?: ChatTool[];
  model?: string;
  error?: string;
}

export const HISTORY_KEY = 'wm.chat.history';
/** Entries kept in storage; the engine sees at most the last 20 messages. */
export const MAX_STORED = 60;
export const MAX_SENT = 20;

/** Applies one streamed event to the assistant entry being written. */
export function applyEvent(entry: ChatEntry, event: ChatEvent): ChatEntry {
  switch (event.type) {
    case 'text':
      return { ...entry, content: entry.content + (event.delta ?? '') };
    case 'tool': {
      const name = event.tool ?? 'tool';
      const status = event.status ?? 'running';
      const tools = [...(entry.tools ?? [])];
      const index = tools.findLastIndex((tool) => tool.name === name && tool.status === 'running');
      if (status !== 'running' && index >= 0) tools[index] = { name, status };
      else tools.push({ name, status });
      return { ...entry, tools };
    }
    case 'done':
      return { ...entry, model: event.model ?? entry.model };
    case 'error':
      return { ...entry, error: event.message || 'The assistant failed. Try again.' };
    default:
      return entry;
  }
}

/**
 * The conversation sent to the engine: every finished turn including the
 * newest user message, without failed or empty replies, newest last.
 */
export function toMessages(entries: ChatEntry[]): ChatMessage[] {
  return entries
    .filter((entry) => entry.content.trim() !== '' && !(entry.role === 'assistant' && entry.error))
    .map((entry) => ({ role: entry.role, content: entry.content }))
    .slice(-MAX_SENT);
}

/**
 * The history kept when the message at index is answered again: everything up
 * to the user message it is or replies to. Later messages are dropped. Null
 * when there is no such user message.
 */
export function retryFrom(entries: ChatEntry[], index: number): ChatEntry[] | null {
  const entry = entries[index];
  if (!entry) return null;
  const end = entry.role === 'user' ? index + 1 : index;
  if (entries[end - 1]?.role !== 'user') return null;
  return entries.slice(0, end);
}

/** How many messages answering the message at index again removes. */
export function removedByRetry(entries: ChatEntry[], index: number): number {
  const kept = retryFrom(entries, index);
  return kept ? entries.length - kept.length : 0;
}

/**
 * The history with one message rewritten. Its error is cleared so an edited
 * reply counts as context again. Null for empty text or an unknown message.
 */
export function editEntry(entries: ChatEntry[], index: number, content: string): ChatEntry[] | null {
  const text = content.trim();
  const entry = entries[index];
  if (!text || !entry) return null;
  const edited: ChatEntry = { ...entry, content: text };
  delete edited.error;
  return entries.with(index, edited);
}

function isEntry(item: unknown): item is ChatEntry {
  if (typeof item !== 'object' || item === null) return false;
  const { role, content } = item as Record<string, unknown>;
  return (role === 'user' || role === 'assistant') && typeof content === 'string';
}

/** Parses stored history, dropping anything malformed. */
export function parseHistory(raw: string | null): ChatEntry[] {
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isEntry).slice(-MAX_STORED);
  } catch {
    return [];
  }
}

const SUGGESTIONS: Record<string, string[]> = {
  '/': ['How did my net worth change this month?', 'What should I watch out for this month?'],
  '/transactions': ['What did I spend the most on this week?', 'Show my largest payments this month.'],
  '/budget': ['Which categories am I over budget on?', 'Where can I cut spending to save more?'],
  '/accounts': ['Summarise my account balances.', 'How much cash do I have available?'],
  '/accounts/credit-cards': ['Which card should I use for groceries?', 'How much credit am I using?'],
  '/accounts/loans': ['Which loan should I pay down first?', 'How much debt is left this year?'],
  '/accounts/goals': ['Am I on track for my goals?', 'How much should I save monthly for my goals?'],
  '/investments': ['How is my portfolio allocated?', 'What are my biggest positions?'],
  '/health': ['How can I improve my financial health score?', 'Is my emergency fund large enough?'],
};

const DEFAULT_SUGGESTIONS = ['How am I doing financially this month?', 'What are my recent transactions?'];

/** Suggested questions for the current page (WM-AI-11). */
export function suggestionsFor(path: string): string[] {
  const match = Object.keys(SUGGESTIONS)
    .filter((key) => path === key || (key !== '/' && path.startsWith(`${key}/`)))
    .sort((a, b) => b.length - a.length)[0];
  return match ? SUGGESTIONS[match] : DEFAULT_SUGGESTIONS;
}
