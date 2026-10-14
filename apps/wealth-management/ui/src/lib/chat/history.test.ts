import { describe, expect, it } from 'vitest';
import {
  applyEvent,
  editEntry,
  MAX_SENT,
  parseHistory,
  removedByRetry,
  suggestionsFor,
  toMessages,
  retryFrom,
  type ChatEntry,
} from './history';

describe('applyEvent', () => {
  it('appends text deltas and tracks tool status', () => {
    let entry: ChatEntry = { role: 'assistant', content: '' };
    entry = applyEvent(entry, { type: 'tool', tool: 'getLoans', status: 'running' });
    entry = applyEvent(entry, { type: 'tool', tool: 'getLoans', status: 'ok' });
    entry = applyEvent(entry, { type: 'text', delta: 'Hel' });
    entry = applyEvent(entry, { type: 'text', delta: 'lo' });
    entry = applyEvent(entry, { type: 'done', model: 'm1' });
    expect(entry).toEqual({
      role: 'assistant',
      content: 'Hello',
      tools: [{ name: 'getLoans', status: 'ok' }],
      model: 'm1',
    });
  });

  it('records errors with a fallback message', () => {
    expect(applyEvent({ role: 'assistant', content: 'Hel' }, { type: 'error' }).error).toBe(
      'The assistant failed. Try again.',
    );
  });
});

describe('toMessages', () => {
  it('includes the newest user message and drops failed or empty replies', () => {
    const entries: ChatEntry[] = [
      { role: 'user', content: 'first' },
      { role: 'assistant', content: 'partial', error: 'boom' },
      { role: 'user', content: 'second' },
      { role: 'assistant', content: '' },
    ];
    expect(toMessages(entries)).toEqual([
      { role: 'user', content: 'first' },
      { role: 'user', content: 'second' },
    ]);
  });

  it('caps the conversation length', () => {
    const entries: ChatEntry[] = Array.from({ length: 30 }, (_, i) => ({ role: 'user', content: String(i) }));
    const messages = toMessages(entries);
    expect(messages).toHaveLength(MAX_SENT);
    expect(messages.at(-1)?.content).toBe('29');
  });
});

describe('retryFrom', () => {
  const entries: ChatEntry[] = [
    { role: 'user', content: 'first' },
    { role: 'assistant', content: 'one' },
    { role: 'user', content: 'second' },
    { role: 'assistant', content: '', error: 'boom' },
  ];

  it('keeps the history up to the user message being answered again', () => {
    expect(retryFrom(entries, 3)).toEqual(entries.slice(0, 3));
    expect(retryFrom(entries, 1)).toEqual(entries.slice(0, 1));
    expect(retryFrom(entries, 2)).toEqual(entries.slice(0, 3));
    expect(retryFrom(entries, 0)).toEqual(entries.slice(0, 1));
  });

  it('counts the messages a retry removes', () => {
    expect(removedByRetry(entries, 3)).toBe(1);
    expect(removedByRetry(entries, 1)).toBe(3);
    expect(removedByRetry(entries, 2)).toBe(1);
    expect(removedByRetry(entries.slice(0, 3), 2)).toBe(0);
    expect(removedByRetry(entries, 4)).toBe(0);
  });

  it('refuses when there is no user message to answer', () => {
    expect(retryFrom([{ role: 'assistant', content: 'hello' }], 0)).toBeNull();
    expect(retryFrom(entries, 4)).toBeNull();
    expect(retryFrom(entries, -1)).toBeNull();
  });
});

describe('editEntry', () => {
  it('rewrites one message in place and keeps the rest of the history', () => {
    const entries: ChatEntry[] = [
      { role: 'user', content: 'first' },
      { role: 'assistant', content: 'partial', error: 'boom', model: 'm1' },
      { role: 'user', content: 'second' },
    ];
    const edited = editEntry(entries, 1, '  fixed  ');
    expect(edited).toEqual([
      { role: 'user', content: 'first' },
      { role: 'assistant', content: 'fixed', model: 'm1' },
      { role: 'user', content: 'second' },
    ]);
    // The rewritten reply is sent as context again once its error is gone.
    expect(toMessages(edited ?? [])).toHaveLength(3);
    expect(entries[1].content).toBe('partial');
  });

  it('refuses empty text and unknown messages', () => {
    const entries: ChatEntry[] = [{ role: 'user', content: 'hi' }];
    expect(editEntry(entries, 0, '   ')).toBeNull();
    expect(editEntry(entries, 1, 'x')).toBeNull();
  });
});

describe('parseHistory', () => {
  it('ignores malformed storage', () => {
    expect(parseHistory('not json')).toEqual([]);
    expect(parseHistory('{"role":"user"}')).toEqual([]);
    expect(parseHistory('[{"role":"user","content":"hi"},{"role":"system","content":"x"},1]')).toEqual([
      { role: 'user', content: 'hi' },
    ]);
  });
});

describe('suggestionsFor', () => {
  it('uses the most specific page', () => {
    expect(suggestionsFor('/accounts/loans')[0]).toContain('loan');
    expect(suggestionsFor('/accounts/loans/x')[0]).toContain('loan');
    expect(suggestionsFor('/budget')[0]).toContain('budget');
    expect(suggestionsFor('/unknown')).toHaveLength(2);
  });
});
