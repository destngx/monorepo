import { describe, expect, it } from 'vitest';
import { parseAnsi, plainOf } from './ansi';
import { paragraphs, tidyLines, toBlocks } from './screen';

const E = '\x1b[';
// Claude Code's own codes: your message on a grey background, its reply after
// a white dot, a tool call after a green dot, and the scrollbar on the right.
const USER_BG = `${E}48;2;55;55;55m`;
const PANEL_BG = `${E}48;2;38;38;38m`;
// Claude Code's side panel, from column 104: a scrollbar, a close mark, a menu.
const SCROLLBAR = `${E}0m${E}2m${PANEL_BG}│${E}0m${PANEL_BG}          `;
const CLOSE = `${E}0m${E}38;2;153;153;153m${PANEL_BG}✕${E}0m${PANEL_BG} ${E}0m`;
const MENU = `${PANEL_BG}  source: Current${E}38;2;153;153;153m${PANEL_BG} ▾${E}0m`;
const pad = (text: string) => text.padEnd(104);
const REPLY_DOT = `${E}0m${E}38;2;255;255;255m⏺${E}0m `;
const TOOL_DOT = `${E}0m${E}38;2;78;186;101m⏺${E}0m `;

const tidy = (text: string) => tidyLines(parseAnsi(text)).map(plainOf);

describe('tidyLines', () => {
  it('drops trailing spaces and blank lines at both ends', () => {
    expect(tidy('\n\n  first   \nsecond\t\n\n\n')).toEqual(['  first', 'second']);
  });

  it('keeps at most one blank line in a row', () => {
    expect(tidy('a\n\n\n\nb')).toEqual(['a', '', 'b']);
  });

  it('removes the side panel Claude Code draws on the right', () => {
    const screen = [pad('text here') + SCROLLBAR, pad('') + SCROLLBAR, pad('more') + CLOSE, pad('next') + MENU];
    expect(tidy(screen.join('\n'))).toEqual(['text here', '', 'more', 'next']);
  });

  it('keeps a background that starts near the left, such as a code diff', () => {
    expect(tidy(`     ${E}48;2;2;40;0m 15 + if x {${E}0m`)).toEqual(['      15 + if x {']);
  });

  it('keeps a table border that is not the scrollbar', () => {
    expect(tidy('│ a │ b │')).toEqual(['│ a │ b │']);
  });

  it('shortens a terminal-wide rule so it does not wrap on a phone', () => {
    expect(tidy(`above\n${'─'.repeat(180)}\nbelow`)).toEqual(['above', '─'.repeat(16), 'below']);
    expect(tidy(`${E}2m${'-'.repeat(40)}${E}0m`)).toEqual(['-'.repeat(16)]);
  });

  it('leaves short rules and text with dashes alone', () => {
    expect(tidy('──── done\na - b - c')).toEqual(['──── done', 'a - b - c']);
  });
});

describe('toBlocks', () => {
  const blocks = (text: string) =>
    toBlocks(tidyLines(parseAnsi(text))).map((b) => [b.kind, b.lines.map(plainOf).join('\n')]);

  it('tells your messages, replies, tool calls and the rest apart', () => {
    const screen = [
      `${USER_BG}❯ fix the login form${E}0m${USER_BG}   `,
      `${USER_BG}  and add a test${E}0m`,
      '',
      `${REPLY_DOT}I'll look at the form first.`,
      '  It has two fields.',
      '',
      '  Both are required.',
      '',
      `${TOOL_DOT}${E}1mRead${E}0m(form.ts)`,
      `${E}38;2;153;153;153m  ⎿  Read 40 lines${E}0m`,
      '',
      `${E}38;2;153;153;153m✻ Worked for 2m${E}0m`,
      '────────────────────────────────────────',
      '❯ ',
      `  ${E}38;2;177;185;249mmonorepo${E}0m | 54%`,
    ].join('\n');
    expect(blocks(screen)).toEqual([
      ['user', 'fix the login form\nand add a test'],
      ['reply', "I'll look at the form first.\nIt has two fields.\n\nBoth are required."],
      ['tool', '⏺ Read(form.ts)\n  ⎿  Read 40 lines'],
      ['terminal', `✻ Worked for 2m\n${'─'.repeat(16)}\n❯\n  monorepo | 54%`],
    ]);
  });

  it('treats an empty prompt line as terminal, not a message', () => {
    expect(blocks(`${USER_BG}❯ ${E}0m`)).toEqual([['terminal', '❯']]);
  });

  it('treats plain indented text at the top as a reply that scrolled off', () => {
    const screen = ['  the rest of a reply', '  - with a list', '', `${E}38;2;153;153;153m✻ Worked for 2m${E}0m`].join(
      '\n',
    );
    expect(blocks(screen)).toEqual([
      ['reply', 'the rest of a reply\n- with a list'],
      ['terminal', '✻ Worked for 2m'],
    ]);
  });

  it('keeps grey indented text at the top as terminal output', () => {
    expect(blocks(`${E}38;2;153;153;153m  ⎿  Read 40 lines${E}0m`)).toEqual([['terminal', '  ⎿  Read 40 lines']]);
  });

  it('shows output from other agents as terminal text', () => {
    expect(blocks('$ ls\nREADME.md')).toEqual([['terminal', '$ ls\nREADME.md']]);
  });
});

describe('paragraphs', () => {
  const join = (text: string) => paragraphs(parseAnsi(text)).map((p) => [p.indent, p.hang, plainOf(p.line)]);

  it('joins lines the terminal wrapped, so the phone can wrap them', () => {
    expect(join('The first part\nand the rest.\n\nNext paragraph.')).toEqual([
      [0, 0, 'The first part and the rest.'],
      [0, 0, ''],
      [0, 0, 'Next paragraph.'],
    ]);
  });

  it('starts a new paragraph at each list item and keeps its hanging indent', () => {
    expect(join('- one that\n  wraps\n- two\n  1. nested')).toEqual([
      [0, 2, '- one that wraps'],
      [0, 2, '- two'],
      [2, 3, '1. nested'],
    ]);
  });

  it('keeps styles across the join', () => {
    const [p] = paragraphs(parseAnsi(`run ${E}38;2;177;185;249mnx${E}0m\nnow`));
    expect(p.line).toEqual([
      { text: 'run ', style: {} },
      { text: 'nx', style: { fg: 'rgb(177 185 249)' } },
      { text: ' now', style: {} },
    ]);
  });
});
