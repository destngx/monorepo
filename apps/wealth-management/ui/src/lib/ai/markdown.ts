// A small markdown renderer for model output. Everything is HTML-escaped
// before any markup is added, so the result is safe for {@html}; only a fixed
// set of tags is produced and links must be http(s).

const escapes: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };

export function escapeHTML(text: string): string {
  return text.replace(/[&<>"']/g, (char) => escapes[char]);
}

/** Renders inline markup of one escaped line: code, bold, italic, links. */
function inline(text: string): string {
  const codes: string[] = [];
  // \uE000 marks code placeholders, so it is dropped from the input first.
  let html = escapeHTML(text.replaceAll('\uE000', '')).replace(/`([^`]+)`/g, (_, code: string) => {
    codes.push(`<code>${code}</code>`);
    return `\uE000${codes.length - 1}\uE000`;
  });
  html = html
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/__(.+?)__/g, '<strong>$1</strong>')
    .replace(/(^|[^*\w])\*(?!\s)(.+?)(?<!\s)\*(?!\*)/g, '$1<em>$2</em>')
    .replace(/(^|[^_\w])_(?!\s)(.+?)(?<!\s)_(?!\w)/g, '$1<em>$2</em>')
    .replace(
      /\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g,
      (_, label: string, href: string) => `<a href="${href}" target="_blank" rel="noopener noreferrer">${label}</a>`,
    );
  return html.replace(/\uE000(\d+)\uE000/g, (_, index: string) => codes[Number(index)]);
}

type Block = { kind: 'p' | 'ul' | 'ol'; lines: string[] };

export function renderMarkdown(markdown: string): string {
  const out: string[] = [];
  let block: Block | null = null;
  let fence: string[] | null = null;

  const flush = () => {
    if (!block) return;
    if (block.kind === 'p') out.push(`<p>${block.lines.map(inline).join('<br>')}</p>`);
    else out.push(`<${block.kind}>${block.lines.map((line) => `<li>${inline(line)}</li>`).join('')}</${block.kind}>`);
    block = null;
  };
  const add = (kind: Block['kind'], line: string) => {
    if (block?.kind === kind) {
      block.lines.push(line);
      return;
    }
    flush();
    block = { kind, lines: [line] };
  };

  for (const raw of markdown.replace(/\r\n?/g, '\n').split('\n')) {
    if (fence) {
      if (/^\s*```/.test(raw)) {
        out.push(`<pre><code>${escapeHTML(fence.join('\n'))}</code></pre>`);
        fence = null;
      } else fence.push(raw);
      continue;
    }
    const line = raw.trim();
    let match: RegExpMatchArray | null;
    if (line.startsWith('```')) {
      flush();
      fence = [];
    } else if (!line) {
      flush();
    } else if ((match = line.match(/^(#{1,6})\s+(.*)$/))) {
      flush();
      const level = Math.min(match[1].length + 2, 6);
      out.push(`<h${level}>${inline(match[2])}</h${level}>`);
    } else if (/^([-*_])\1{2,}$/.test(line)) {
      flush();
      out.push('<hr>');
    } else if ((match = line.match(/^[-*+]\s+(.*)$/))) {
      add('ul', match[1]);
    } else if ((match = line.match(/^\d+[.)]\s+(.*)$/))) {
      add('ol', match[1]);
    } else if ((match = line.match(/^>\s?(.*)$/))) {
      add('p', match[1]);
    } else {
      add('p', line);
    }
  }
  if (fence) out.push(`<pre><code>${escapeHTML(fence.join('\n'))}</code></pre>`);
  flush();
  return out.join('');
}
