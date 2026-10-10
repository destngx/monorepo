import { describe, expect, it } from 'vitest';
import { escapeSvelte, highlightCode } from './highlight.js';

describe('escapeSvelte', () => {
  it('neutralises Svelte template characters', () => {
    expect(escapeSvelte('{a} `b` \\c')).toBe('&#123;a&#125; &#96;b&#96; &#92;c');
  });
});

describe('highlightCode', () => {
  it('wraps highlighted html in an @html block without raw braces', async () => {
    const out = await highlightCode('const x = { a: 1 };', 'ts');

    expect(out.startsWith('{@html `')).toBe(true);
    expect(out.slice(8, -2)).not.toMatch(/[{}]/);
    expect(out).toContain('shiki');
  });

  it('falls back to plain text for unknown languages', async () => {
    await expect(highlightCode('hello', 'klingon')).resolves.toContain('hello');
  });
});
