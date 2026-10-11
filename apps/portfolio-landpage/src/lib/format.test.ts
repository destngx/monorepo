import { describe, expect, it } from 'vitest';
import { formatDate, readingTime } from './format';

describe('formatDate', () => {
  it('formats dates in a stable, timezone-independent way', () => {
    expect(formatDate(new Date('2026-03-05T00:00:00Z'))).toBe('5 Mar 2026');
  });
});

describe('readingTime', () => {
  it('rounds up to whole minutes at 220 wpm with a one-minute floor', () => {
    expect(readingTime('word '.repeat(10))).toBe(1);
    expect(readingTime('word '.repeat(450))).toBe(3);
  });
});
