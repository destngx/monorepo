import { describe, expect, it } from 'vitest';
import { sourceLabel, sourceOf, sourceState, sourceTitle } from './source';

const asOf = '2026-10-13T08:00:00Z';

describe('sourceState', () => {
  it('is live when the first provider answered', () => {
    expect(sourceState({ provider: 'vnstock', asOf })).toBe('live');
    expect(sourceState({ provider: 'vnstock', fallback: [] })).toBe('live');
  });

  it('is fallback when a later provider answered', () => {
    expect(sourceState({ provider: 'yahoo', fallback: ['vnstock: quota'] })).toBe('fallback');
  });

  it('is stale when every provider failed, even after a fallback', () => {
    expect(sourceState({ provider: 'fmarket', fallback: ['vnstock: quota'], stale: true, asOf })).toBe('stale');
  });
});

describe('sourceLabel', () => {
  it('names the provider and an optional detail', () => {
    expect(sourceLabel({ provider: 'fmarket', asOf })).toBe('fmarket');
    expect(sourceLabel({ provider: 'fmarket', asOf }, 'bank sell')).toBe('fmarket · bank sell');
  });

  it('adds the age of a stale copy', () => {
    const label = sourceLabel({ provider: 'fmarket', stale: true, asOf });
    expect(label).toMatch(/^fmarket · as of \d{2}\/10\/2026 \d{2}:00$/);
  });
});

describe('sourceTitle', () => {
  it('says who served the data and when it was read', () => {
    expect(sourceTitle({ provider: 'vnstock', asOf })).toMatch(/^Served by vnstock, read 13\/10\/2026 \d{2}:00\.$/);
    expect(sourceTitle({ provider: 'fmarket' }, 'bank sell')).toBe('Served by fmarket (bank sell).');
  });

  it('lists the providers that failed first', () => {
    expect(sourceTitle({ provider: 'yahoo', fallback: ['vnstock: quota', 'fmarket: down'] })).toBe(
      'Served by yahoo.\nFailed first: vnstock: quota; fmarket: down',
    );
  });

  it('warns that a stale copy is the last good one', () => {
    expect(sourceTitle({ provider: 'fmarket', stale: true, asOf })).toMatch(
      /^Every provider failed\. Showing the last good copy from fmarket, read 13\/10\/2026/,
    );
  });
});

describe('sourceOf', () => {
  it('reads the source of an fmarket payload', () => {
    const payload = { data: [], source: { provider: 'fmarket', fallback: ['x: y', 3], stale: true, asOf } };
    expect(sourceOf(payload)).toEqual({ provider: 'fmarket', fallback: ['x: y'], stale: true, asOf });
  });

  it('ignores payloads without a usable source', () => {
    expect(sourceOf(null)).toBeUndefined();
    expect(sourceOf([])).toBeUndefined();
    expect(sourceOf({ data: [] })).toBeUndefined();
    expect(sourceOf({ source: { provider: '' } })).toBeUndefined();
    expect(sourceOf({ source: 'fmarket' })).toBeUndefined();
  });
});
