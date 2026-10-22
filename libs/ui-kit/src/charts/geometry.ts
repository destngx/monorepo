// Pure geometry for the hand-written SVG charts.

export type Extent = [number, number];
export type Point = [number, number];

const round = (value: number) => Math.round(value * 100) / 100;

export interface ExtentOptions {
  /** Stretch the extent to 0, for bars that grow from a zero baseline. */
  includeZero?: boolean;
  /** Pad both ends by this share of the span, so marks clear the edges. */
  pad?: number;
}

/** The values' extent. A flat one is padded by 10% of its value, so the line sits in the middle. */
export function extent(values: number[], { includeZero = false, pad = 0 }: ExtentOptions = {}): Extent {
  if (values.length === 0) return [0, 1];
  let min = Math.min(...values);
  let max = Math.max(...values);
  if (includeZero) {
    min = Math.min(min, 0);
    max = Math.max(max, 0);
  }
  if (min === max) {
    const flat = Math.abs(min) * 0.1 || 1;
    return [min - flat, max + flat];
  }
  const p = (max - min) * pad;
  return [min - p, max + p];
}

export function scale([d0, d1]: Extent, [r0, r1]: Extent): (value: number) => number {
  const span = d1 - d0 || 1;
  return (value) => r0 + ((value - d0) / span) * (r1 - r0);
}

/** Round tick values inside the extent, at most `count` steps apart from first to last. */
export function ticks([min, max]: Extent, count = 4): number[] {
  const span = max - min;
  if (span <= 0) return [min];
  const rough = span / count;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const step = [1, 2, 5, 10].map((m) => m * magnitude).find((s) => span / s <= count) ?? magnitude * 10;
  const result: number[] = [];
  for (let value = Math.ceil(min / step - 1e-9) * step; value <= max + step * 1e-9; value += step) {
    result.push(Math.round(value / step) * step);
  }
  // Drops float noise such as 0.30000000000000004.
  return result.map((value) => Number(value.toPrecision(12)));
}

/** Days since 1970-01-01 of a yyyy-MM-dd date, from its calendar part so no time zone shifts it. */
export function toDay(iso: string): number {
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  return Date.UTC(y, m - 1, d) / 86_400_000;
}

/** The yyyy-MM-dd date of a day number. */
export function fromDay(day: number): string {
  return new Date(Math.round(day) * 86_400_000).toISOString().slice(0, 10);
}

/** `count` evenly spread day numbers from first to last, both included. */
export function dateTicks(first: number, last: number, count = 4): number[] {
  if (last <= first || count < 2) return [first];
  return Array.from({ length: count }, (_, i) => Math.round(first + ((last - first) * i) / (count - 1)));
}

/**
 * Points of a line chart scaled into width × height (y grows downward). The x
 * positions are `x` (e.g. day numbers) or, without it, the values' indexes.
 */
export function linePoints(values: number[], width: number, height: number, padding = 2, x?: number[]): Point[] {
  if (values.length === 0) return [];
  const xs = x ?? values.map((_, index) => index);
  const first = xs[0];
  const last = xs[xs.length - 1];
  const sx = scale([first, last > first ? last : first + 1], [padding, width - padding]);
  const sy = scale(extent(values), [height - padding, padding]);
  return values.map((value, index) => [round(values.length === 1 ? width / 2 : sx(xs[index])), round(sy(value))]);
}

export function linePath(points: Point[]): string {
  return points.map(([x, y], index) => `${index === 0 ? 'M' : 'L'}${round(x)},${round(y)}`).join(' ');
}

/** Closes a line path down to the baseline for an area fill. */
export function areaPath(points: Point[], height: number): string {
  if (points.length === 0) return '';
  const first = points[0];
  const last = points[points.length - 1];
  return `${linePath(points)} L${round(last[0])},${height} L${round(first[0])},${height} Z`;
}

/** Annular sector from start to end angle (radians, 0 = 12 o'clock, clockwise). */
export function arcPath(cx: number, cy: number, outer: number, inner: number, start: number, end: number): string {
  // A full circle cannot be drawn as one arc; stop just short of it.
  const sweep = Math.min(end - start, Math.PI * 2 - 1e-4);
  const finish = start + sweep;
  const point = (radius: number, angle: number) => [
    round(cx + radius * Math.sin(angle)),
    round(cy - radius * Math.cos(angle)),
  ];
  const large = sweep > Math.PI ? 1 : 0;
  const [x0, y0] = point(outer, start);
  const [x1, y1] = point(outer, finish);
  const [x2, y2] = point(inner, finish);
  const [x3, y3] = point(inner, start);
  return `M${x0},${y0} A${outer},${outer} 0 ${large} 1 ${x1},${y1} L${x2},${y2} A${inner},${inner} 0 ${large} 0 ${x3},${y3} Z`;
}

/** Splits a full circle by value; zero or negative values get no slice. */
export function slices(values: number[]): { start: number; end: number; fraction: number }[] {
  const total = values.reduce((sum, value) => sum + Math.max(value, 0), 0);
  let angle = 0;
  return values.map((value) => {
    const fraction = total > 0 ? Math.max(value, 0) / total : 0;
    const start = angle;
    angle += fraction * Math.PI * 2;
    return { start, end: angle, fraction };
  });
}

/** Vertices of a radar polygon; fractions are clamped to 0..1. */
export function radarPoints(fractions: number[], cx: number, cy: number, radius: number): Point[] {
  const step = (Math.PI * 2) / Math.max(fractions.length, 1);
  return fractions.map((fraction, index) => {
    const r = radius * Math.min(Math.max(fraction, 0), 1);
    const angle = index * step;
    return [round(cx + r * Math.sin(angle)), round(cy - r * Math.cos(angle))];
  });
}
