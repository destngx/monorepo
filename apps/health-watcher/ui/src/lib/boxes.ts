import type { Point, Quad } from './api/types';

const clamp = (n: number) => Math.min(1, Math.max(0, n));

/** The box a drag from a to b draws, corners clockwise from the top-left, inside the page. */
export function dragQuad(a: Point, b: Point): Quad {
  const left = clamp(Math.min(a.x, b.x));
  const right = clamp(Math.max(a.x, b.x));
  const top = clamp(Math.min(a.y, b.y));
  const bottom = clamp(Math.max(a.y, b.y));
  return [
    { x: left, y: top },
    { x: right, y: top },
    { x: right, y: bottom },
    { x: left, y: bottom },
  ];
}

/** Whether a drawn box covers some area; a plain click draws none. */
export function usableBox(q: Quad): boolean {
  return q[1].x - q[0].x >= 0.005 && q[2].y - q[1].y >= 0.005;
}

/** A quad's corners scaled to an image, as SVG polygon points. */
export function quadPoints(q: Quad, width: number, height: number): string {
  return q.map((p) => `${Math.round(p.x * width)},${Math.round(p.y * height)}`).join(' ');
}

/** A short key that changes when the boxes change, for cache-busting image URLs. */
export function boxesKey(boxes: Quad[]): string {
  let hash = 5381;
  for (const ch of JSON.stringify(boxes)) hash = ((hash << 5) + hash + ch.charCodeAt(0)) | 0;
  return (hash >>> 0).toString(36);
}
