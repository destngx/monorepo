import { describe, expect, it } from 'vitest';
import { applyEvent, MAX_SENT, parseHistory, suggestionsFor, toMessages, type ChatEntry } from './history';

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
