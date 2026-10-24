import { plainOf, type Line, type Span } from './ansi';

/** A rule longer than this many characters is shortened to it. */
const RULE_WIDTH = 16;
const RULE = /^([─━═\-_])\1{19,}$/u;

/**
 * Makes an agent's terminal output readable at phone width. The engine sends
 * unwrapped lines, which the page wraps. This trims the padding a terminal
 * leaves, removes the scrollbar Claude Code draws on the right, shortens
 * full-width rules, and keeps at most one blank line in a row.
 */
export function tidyLines(lines: Line[]): Line[] {
  const out: Line[] = [];
  for (const raw of lines) {
    const line = shortenRule(trimEnd(dropSidePanel(raw)));
    const isBlank = line.length === 0;
    if (isBlank && (out.length === 0 || out[out.length - 1].length === 0)) continue;
    out.push(line);
  }
  while (out.length > 0 && out[out.length - 1].length === 0) out.pop();
  return out;
}

/** A run of one background colour starting at or past this column is a side panel. */
const PANEL_COLUMN = 40;

/**
 * Claude Code draws a side panel on the right (a scrollbar, a close mark, a
 * menu), all on one background colour. Drops a trailing run of spans that
 * share a background and start far from the left edge. Your messages, whose
 * background starts at column 0, and code diffs, near the left, are kept.
 */
function dropSidePanel(line: Line): Line {
  const bg = line[line.length - 1]?.style.bg;
  if (!bg) return line;
  let start = line.length - 1;
  while (start > 0 && line[start - 1].style.bg === bg) start--;
  const column = line.slice(0, start).reduce((n, s) => n + s.text.length, 0);
  return column >= PANEL_COLUMN ? line.slice(0, start) : line;
}

function trimEnd(line: Line): Line {
  const out = [...line];
  while (out.length > 0) {
    const last = out[out.length - 1];
    const text = last.text.trimEnd();
    if (text) {
      out[out.length - 1] = { ...last, text };
      break;
    }
    out.pop();
  }
  return out;
}

function shortenRule(line: Line): Line {
  const plain = plainOf(line).trim();
  const rule = RULE.exec(plain);
  if (!rule) return line;
  return [{ text: rule[1].repeat(RULE_WIDTH), style: line.find((s) => s.text.trim())?.style ?? {} }];
}

/** What a run of lines is: your message, the agent's reply, a tool call, or other terminal output. */
export type BlockKind = 'user' | 'reply' | 'tool' | 'terminal';

export interface Block {
  kind: BlockKind;
  lines: Line[];
}

// Claude Code draws a reply's dot in white or the default colour; tool calls get a green, red or grey dot.
const REPLY_DOT_COLOURS = new Set([undefined, 'rgb(255 255 255)']);
// The lavender of inline code in a reply.
const INLINE_CODE = 'rgb(177 185 249)';

/**
 * Groups Claude Code's screen into blocks. Your messages start with "❯" on a
 * background colour; replies and tool calls start with "⏺" and continue on
 * indented lines. Output from any other agent stays terminal text.
 */
export function toBlocks(lines: Line[]): Block[] {
  const blocks: Block[] = [];
  let i = 0;

  // The screen often starts inside a reply whose first line scrolled off: an
  // indented run in the default colour. Grey indented text is tool output.
  if (lines.length > 0 && !startsBlock(lines[0]) && lines[0].length > 0) {
    let end = 0;
    while (end < lines.length && !startsBlock(lines[end])) end++;
    const run = lines.slice(0, end);
    if (isPlainText(run)) {
      const group = run.map((line) => dedent(line, 2));
      while (group.length > 1 && group[group.length - 1].length === 0) group.pop();
      blocks.push({ kind: 'reply', lines: group });
      i = end;
    }
  }

  while (i < lines.length) {
    const line = lines[i];
    const plain = plainOf(line);
    const bg = line[0]?.style.bg;

    if (plain.startsWith('❯') && bg && plain.slice(1).trim() !== '') {
      const group = [dropPrefix(line, plain.length - plain.slice(1).trimStart().length)];
      i++;
      while (i < lines.length && lines[i].length > 0 && lines[i][0].style.bg === bg) {
        group.push(dedent(lines[i], 2));
        i++;
      }
      blocks.push({ kind: 'user', lines: group });
      continue;
    }

    if (plain.startsWith('⏺')) {
      const dot = line.find((s) => s.text.includes('⏺'));
      const isReply = REPLY_DOT_COLOURS.has(dot?.style.fg);
      const group = [isReply ? dropPrefix(line, plain.length - plain.slice(1).trimStart().length) : line];
      i++;
      while (i < lines.length && !startsBlock(lines[i])) {
        group.push(isReply ? dedent(lines[i], 2) : lines[i]);
        i++;
      }
      while (group.length > 1 && group[group.length - 1].length === 0) group.pop();
      blocks.push({ kind: isReply ? 'reply' : 'tool', lines: group });
      continue;
    }

    const last = blocks[blocks.length - 1];
    if (last?.kind === 'terminal') last.lines.push(line);
    else if (line.length > 0) blocks.push({ kind: 'terminal', lines: [line] });
    i++;
  }
  for (const block of blocks) {
    while (block.lines.length > 1 && block.lines[block.lines.length - 1].length === 0) block.lines.pop();
  }
  return blocks;
}

/** Whether most of the text is in the default or white colour, as a reply's is. */
function isPlainText(lines: Line[]): boolean {
  let plain = 0;
  let total = 0;
  for (const line of lines) {
    for (const span of line) {
      const n = span.text.trim().length;
      total += n;
      if (REPLY_DOT_COLOURS.has(span.style.fg) || span.style.fg === INLINE_CODE) plain += n;
    }
  }
  return total > 0 && plain / total > 0.7;
}

/** A line that starts at the left edge starts a new block; indented and blank lines continue one. */
function startsBlock(line: Line): boolean {
  const plain = plainOf(line);
  return plain !== '' && plain[0] !== ' ';
}

/** The line without its first n characters. */
function dropPrefix(line: Line, n: number): Line {
  const out: Line = [];
  let left = n;
  for (const span of line) {
    if (left >= span.text.length) {
      left -= span.text.length;
      continue;
    }
    out.push(left > 0 ? { ...span, text: span.text.slice(left) } : span);
    left = 0;
  }
  return out;
}

/** The line without up to n leading spaces. */
function dedent(line: Line, n: number): Line {
  const spaces = plainOf(line).match(/^ */)?.[0].length ?? 0;
  return dropPrefix(line, Math.min(n, spaces));
}

export interface Paragraph {
  /** Leading spaces of the first line, in characters. */
  indent: number;
  /** Width of a list marker ("- ", "1. "), so wrapped lines line up after it. */
  hang: number;
  line: Line;
}

const LIST_ITEM = /^([-*•]|\d+[.)]) /;

/**
 * Joins the lines a terminal wrapped back into paragraphs, so a reply wraps
 * to the phone's width. A blank line or a list item starts a new paragraph.
 */
export function paragraphs(lines: Line[]): Paragraph[] {
  const out: Paragraph[] = [];
  let open: Paragraph | null = null;
  for (const raw of lines) {
    const plain = plainOf(raw);
    const indent = plain.match(/^ */)?.[0].length ?? 0;
    const line = dropPrefix(raw, indent);
    const text = plain.slice(indent);
    if (text === '') {
      out.push({ indent: 0, hang: 0, line: [] });
      open = null;
      continue;
    }
    const item = LIST_ITEM.exec(text);
    if (open && !item) {
      open.line = joinLines(open.line, line);
      continue;
    }
    open = { indent, hang: item ? item[0].length : 0, line };
    out.push(open);
  }
  return out;
}

function joinLines(a: Line, b: Line): Line {
  const first: Span | undefined = b[0];
  if (!first) return a;
  return mergeSpans([...a, { ...first, text: ` ${first.text}` }, ...b.slice(1)]);
}

function mergeSpans(line: Line): Line {
  const out: Line = [];
  for (const span of line) {
    const prev = out[out.length - 1];
    if (prev && JSON.stringify(prev.style) === JSON.stringify(span.style)) prev.text += span.text;
    else out.push({ ...span });
  }
  return out;
}
