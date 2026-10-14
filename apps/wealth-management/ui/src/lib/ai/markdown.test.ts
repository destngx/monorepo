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

  it('renders tables with column alignment and inline markup', () => {
    expect(renderMarkdown('| Item | Amount |\n|---|---:|\n| Trip | ₫38 |\n| **Total** | **₫63** |')).toBe(
      '<div class="md-table"><table><thead><tr><th>Item</th><th data-align="right">Amount</th></tr></thead>' +
        '<tbody><tr><td>Trip</td><td data-align="right">₫38</td></tr>' +
        '<tr><td><strong>Total</strong></td><td data-align="right"><strong>₫63</strong></td></tr></tbody></table></div>',
    );
  });

  it('accepts rows without outer pipes, pads short rows and keeps escaped or quoted pipes', () => {
    expect(renderMarkdown('a | b\n:-: | :--\n`x|y` \\| z\n\nafter')).toBe(
      '<div class="md-table"><table><thead><tr><th data-align="center">a</th><th data-align="left">b</th></tr></thead>' +
        '<tbody><tr><td data-align="center"><code>x|y</code> | z</td><td data-align="left"></td></tr></tbody></table></div>' +
        '<p>after</p>',
    );
  });

  it('ends a table at the first line that is not a row', () => {
    expect(renderMarkdown('| a |\n| - |\n| 1 |\ntext')).toBe(
      '<div class="md-table"><table><thead><tr><th>a</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table></div><p>text</p>',
    );
  });

  it('leaves a pipe line without a delimiter row as text', () => {
    expect(renderMarkdown('a | b\nc | d')).toBe('<p>a | b<br>c | d</p>');
  });
});
