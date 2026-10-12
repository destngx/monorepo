import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown';

describe('renderMarkdown', () => {
  it('renders headings, lists, emphasis and code', () => {
    expect(renderMarkdown('## Plan\n- **Cut** coffee\n- use `budget`\n\n1. one\n2. _two_')).toBe(
      '<h4>Plan</h4><ul><li><strong>Cut</strong> coffee</li><li>use <code>budget</code></li></ul><ol><li>one</li><li><em>two</em></li></ol>',
    );
  });

  it('joins paragraph lines and keeps fenced code verbatim', () => {
    expect(renderMarkdown('a\nb\n\n```\n**x** <b>\n```')).toBe('<p>a<br>b</p><pre><code>**x** &lt;b&gt;</code></pre>');
  });

  it('escapes HTML and only links http(s) URLs', () => {
    const html = renderMarkdown('<img src=x onerror=alert(1)> [ok](https://a.vn/x) [bad](javascript:alert(1))');
    expect(html).not.toContain('<img');
    expect(html).toContain('<a href="https://a.vn/x" target="_blank" rel="noopener noreferrer">ok</a>');
    expect(html).not.toContain('href="javascript');
  });

  it('does not treat multiplication or snake_case as emphasis', () => {
    expect(renderMarkdown('2 * 3 * 4 and total_spent_now')).toBe('<p>2 * 3 * 4 and total_spent_now</p>');
  });
});
