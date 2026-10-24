import { describe, expect, it } from 'vitest';
import { parseAnsi, plainOf, xterm256 } from './ansi';

const ESC = '\x1b[';

describe('parseAnsi', () => {
  it('splits lines into styled spans', () => {
    const lines = parseAnsi(`${ESC}1mbold${ESC}0m plain\n${ESC}38;2;177;185;249mcode${ESC}0m`);
    expect(lines).toEqual([
      [
        { text: 'bold', style: { bold: true } },
        { text: ' plain', style: {} },
      ],
      [{ text: 'code', style: { fg: 'rgb(177 185 249)' } }],
    ]);
  });

  it('reads 16, 256 and 24-bit colours, foreground and background', () => {
    const [line] = parseAnsi(`${ESC}31ma${ESC}0m${ESC}38;5;6mb${ESC}0m${ESC}48;2;55;55;55mc${ESC}0m${ESC}97md`);
    expect(line.map((s) => s.style)).toEqual([
      { fg: xterm256(1) },
      { fg: xterm256(6) },
      { bg: 'rgb(55 55 55)' },
      { fg: xterm256(15) },
    ]);
  });

  it('keeps attributes until they are turned off', () => {
    const [line] = parseAnsi(`${ESC}2m${ESC}3mdim italic${ESC}22m italic${ESC}23m plain`);
    expect(line).toEqual([
      { text: 'dim italic', style: { dim: true, italic: true } },
      { text: ' italic', style: { italic: true } },
      { text: ' plain', style: {} },
    ]);
  });

  it('drops carriage returns, cursor moves and window titles', () => {
    const lines = parseAnsi(`a${ESC}2Kb\x1b]0;title\x07c\r`);
    expect(lines.map(plainOf)).toEqual(['abc']);
  });

  it('turns non-breaking spaces into spaces so text wraps', () => {
    expect(plainOf(parseAnsi('❯\u00a0hi')[0])).toBe('❯ hi');
  });
});

describe('xterm256', () => {
  it('maps the cube and the grey ramp', () => {
    expect(xterm256(16)).toBe('rgb(0 0 0)');
    expect(xterm256(196)).toBe('rgb(255 0 0)');
    expect(xterm256(232)).toBe('rgb(8 8 8)');
  });
});
