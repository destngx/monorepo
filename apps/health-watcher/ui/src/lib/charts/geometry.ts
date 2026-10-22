// Pure geometry for the hand-written SVG charts.

export type Extent = [number, number];

const round = (value: number) => Math.round(value * 100) / 100;

/** The values' extent, padded by `pad` of the span so marks clear the edges; a flat one by 10% of the value. */
export function extent(values: number[], pad = 0): Extent {
  if (values.length === 0) return [0, 1];
  const min = Math.min(...values);
  const max = Math.max(...values);
  if (min === max) {
    const p = Math.abs(min) * 0.1 || 1;
    return [min - p, max + p];
  }
  const p = (max - min) * pad;
  return [min - p, max + p];
}

export function scale([d0, d1]: Extent, [r0, r1]: Extent): (value: number) => number {
  const span = d1 - d0 || 1;
  return (value) => r0 + ((value - d0) / span) * (r1 - r0);
}

/** About `count` round tick values inside the extent. */
export function ticks([min, max]: Extent, count = 4): number[] {
  const span = max - min;
  if (span <= 0) return [min];
  const rough = span / count;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const step = [1, 2, 5, 10].map((m) => m * magnitude).find((s) => span / s <= count + 1) ?? magnitude * 10;
  const result: number[] = [];
  for (let value = Math.ceil(min / step - 1e-9) * step; value <= max + step * 1e-9; value += step) {
    result.push(Math.round(value / step) * step);
  }
  return result.map((v) => Number(v.toPrecision(12)));
}

/** Days since 1970-01-01 of a YYYY-MM-DD date, from its calendar part so no time zone shifts it. */
export function toDay(iso: string): number {
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  return Date.UTC(y, m - 1, d) / 86_400_000;
}

/** The YYYY-MM-DD date of a day number. */
export function fromDay(day: number): string {
  return new Date(Math.round(day) * 86_400_000).toISOString().slice(0, 10);
}

/** `count` evenly spread day numbers from first to last, both included. */
export function dateTicks(first: number, last: number, count = 4): number[] {
  if (last <= first || count < 2) return [first];
  return Array.from({ length: count }, (_, i) => Math.round(first + ((last - first) * i) / (count - 1)));
}

export function linePath(points: [number, number][]): string {
  return points.map(([x, y], index) => `${index === 0 ? 'M' : 'L'}${round(x)},${round(y)}`).join(' ');
}
