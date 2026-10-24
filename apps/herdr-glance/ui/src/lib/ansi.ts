/** How one run of terminal text looks. Colours are CSS colour strings. */
export interface Style {
  fg?: string;
  bg?: string;
  bold?: boolean;
  dim?: boolean;
  italic?: boolean;
  underline?: boolean;
  inverse?: boolean;
}

export interface Span {
  text: string;
  style: Style;
}

/** One terminal line as styled runs of text. */
export type Line = Span[];

// The standard 16 colours, as xterm draws them.
const BASIC = [
  [0, 0, 0],
  [205, 49, 49],
  [13, 188, 121],
  [229, 229, 16],
  [36, 114, 200],
  [188, 63, 188],
  [17, 168, 205],
  [229, 229, 229],
  [102, 102, 102],
  [241, 76, 76],
  [35, 209, 139],
  [245, 245, 67],
  [59, 142, 234],
  [214, 112, 214],
  [41, 184, 219],
  [255, 255, 255],
];

function rgb(r: number, g: number, b: number): string {
  return `rgb(${r} ${g} ${b})`;
}

/** The CSS colour of an xterm 256-colour index. */
export function xterm256(n: number): string {
  if (n < 16) return rgb(BASIC[n][0], BASIC[n][1], BASIC[n][2]);
  if (n < 232) {
    const level = (v: number) => (v === 0 ? 0 : 55 + v * 40);
    const i = n - 16;
    return rgb(level(Math.floor(i / 36)), level(Math.floor(i / 6) % 6), level(i % 6));
  }
  const grey = 8 + (n - 232) * 10;
  return rgb(grey, grey, grey);
}

// SGR sequences (colours and attributes), other CSI sequences, and OSC strings.
// Matching the escape and bell characters is the point here.
// eslint-disable-next-line no-control-regex
const ESCAPES = /\x1b\[([0-9;:?]*)([A-Za-z])|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-_]/g;

/** Applies one SGR sequence's parameters to a style. */
function applySGR(style: Style, params: string): Style {
  const codes = params === '' ? [0] : params.split(/[;:]/).map(Number);
  let next: Style = { ...style };
  for (let i = 0; i < codes.length; i++) {
    const c = codes[i];
    if (c === 0) next = {};
    else if (c === 1) next.bold = true;
    else if (c === 2) next.dim = true;
    else if (c === 3) next.italic = true;
    else if (c === 4) next.underline = true;
    else if (c === 7) next.inverse = true;
    else if (c === 22) {
      delete next.bold;
      delete next.dim;
    } else if (c === 23) delete next.italic;
    else if (c === 24) delete next.underline;
    else if (c === 27) delete next.inverse;
    else if (c >= 30 && c <= 37) next.fg = xterm256(c - 30);
    else if (c >= 90 && c <= 97) next.fg = xterm256(c - 90 + 8);
    else if (c === 39) delete next.fg;
    else if (c >= 40 && c <= 47) next.bg = xterm256(c - 40);
    else if (c >= 100 && c <= 107) next.bg = xterm256(c - 100 + 8);
    else if (c === 49) delete next.bg;
    else if (c === 38 || c === 48) {
      const key = c === 38 ? 'fg' : 'bg';
      if (codes[i + 1] === 5 && codes[i + 2] !== undefined) {
        next[key] = xterm256(codes[i + 2]);
        i += 2;
      } else if (codes[i + 1] === 2 && codes[i + 4] !== undefined) {
        next[key] = rgb(codes[i + 2], codes[i + 3], codes[i + 4]);
        i += 4;
      }
    }
  }
  return next;
}

/** Splits terminal text with ANSI codes into lines of styled spans. */
export function parseAnsi(text: string): Line[] {
  return text.split('\n').map((raw) => {
    const line: Line = [];
    let style: Style = {};
    let last = 0;
    const push = (chunk: string) => {
      const clean = chunk.replace(/\r/g, '').replace(/\u00a0/g, ' ');
      if (!clean) return;
      const prev = line[line.length - 1];
      if (prev && sameStyle(prev.style, style)) prev.text += clean;
      else line.push({ text: clean, style });
    };
    for (const match of raw.matchAll(ESCAPES)) {
      push(raw.slice(last, match.index));
      last = match.index + match[0].length;
      if (match[2] === 'm') style = applySGR(style, match[1]);
    }
    push(raw.slice(last));
    return line;
  });
}

function sameStyle(a: Style, b: Style): boolean {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)]) as Set<keyof Style>;
  for (const k of keys) if (a[k] !== b[k]) return false;
  return true;
}

/** A line's text without styles. */
export function plainOf(line: Line): string {
  return line.map((s) => s.text).join('');
}
