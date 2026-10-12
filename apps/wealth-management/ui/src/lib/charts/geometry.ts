// Pure geometry for the hand-written SVG charts.

export type Extent = [number, number];

export function extent(values: number[], includeZero = false): Extent {
  if (values.length === 0) return [0, 1];
  let min = Math.min(...values);
  let max = Math.max(...values);
  if (includeZero) {
    min = Math.min(min, 0);
    max = Math.max(max, 0);
  }
  if (min === max) {
    // Flat series: pad so the line sits in the middle instead of on an edge.
    const pad = Math.abs(min) * 0.1 || 1;
    return [min - pad, max + pad];
  }
  return [min, max];
}

export function scale([d0, d1]: Extent, [r0, r1]: Extent): (value: number) => number {
  const span = d1 - d0 || 1;
  return (value) => r0 + ((value - d0) / span) * (r1 - r0);
}

const round = (value: number) => Math.round(value * 100) / 100;

/** Points of a line chart scaled into width × height (y grows downward). */
export function linePoints(values: number[], width: number, height: number, padding = 2): [number, number][] {
  if (values.length === 0) return [];
  const x = scale([0, Math.max(values.length - 1, 1)], [padding, width - padding]);
  const y = scale(extent(values), [height - padding, padding]);
  return values.map((value, index) => [round(values.length === 1 ? width / 2 : x(index)), round(y(value))]);
}

export function linePath(points: [number, number][]): string {
  return points.map(([x, y], index) => `${index === 0 ? 'M' : 'L'}${x},${y}`).join(' ');
}

/** Closes a line path down to the baseline for an area fill. */
export function areaPath(points: [number, number][], height: number): string {
  if (points.length === 0) return '';
  const first = points[0];
  const last = points[points.length - 1];
  return `${linePath(points)} L${last[0]},${height} L${first[0]},${height} Z`;
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
export function radarPoints(fractions: number[], cx: number, cy: number, radius: number): [number, number][] {
  const step = (Math.PI * 2) / Math.max(fractions.length, 1);
  return fractions.map((fraction, index) => {
    const r = radius * Math.min(Math.max(fraction, 0), 1);
    const angle = index * step;
    return [round(cx + r * Math.sin(angle)), round(cy - r * Math.cos(angle))];
  });
}

/** About `count` round tick values covering the extent. */
export function ticks([min, max]: Extent, count = 4): number[] {
  const span = max - min;
  if (span <= 0) return [min];
  const rough = span / count;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const step = [1, 2, 5, 10].map((m) => m * magnitude).find((s) => span / s <= count) ?? magnitude * 10;
  const result: number[] = [];
  for (let value = Math.ceil(min / step) * step; value <= max + step * 1e-9; value += step) {
    result.push(round(value));
  }
  return result;
}
