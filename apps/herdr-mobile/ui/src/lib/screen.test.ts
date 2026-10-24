import { describe, expect, it } from 'vitest';
import { tidyScreen } from './screen';

describe('tidyScreen', () => {
  it('drops trailing spaces and blank lines at both ends', () => {
    expect(tidyScreen('\n\n  first   \nsecond\t\n\n\n')).toBe('  first\nsecond');
  });

  it('keeps at most one blank line in a row', () => {
    expect(tidyScreen('a\n\n\n\nb')).toBe('a\n\nb');
  });

  it('shortens a terminal-wide rule so it does not wrap on a phone', () => {
    const rule = '─'.repeat(180);
    expect(tidyScreen(`above\n${rule}\nbelow`)).toBe(`above\n${'─'.repeat(16)}\nbelow`);
    expect(tidyScreen('-'.repeat(40))).toBe('-'.repeat(16));
  });

  it('leaves short rules and text with dashes alone', () => {
    expect(tidyScreen('──── done')).toBe('──── done');
    expect(tidyScreen('a - b - c')).toBe('a - b - c');
  });
});
