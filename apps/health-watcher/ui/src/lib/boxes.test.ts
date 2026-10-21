import { describe, expect, it } from 'vitest';
import { boxesKey, dragQuad, quadPoints, usableBox } from './boxes';

describe('dragQuad', () => {
  it('orders the corners whichever way the drag went', () => {
    expect(dragQuad({ x: 0.6, y: 0.5 }, { x: 0.2, y: 0.1 })).toEqual([
      { x: 0.2, y: 0.1 },
      { x: 0.6, y: 0.1 },
      { x: 0.6, y: 0.5 },
      { x: 0.2, y: 0.5 },
    ]);
  });
  it('clamps a drag that leaves the page', () => {
    const q = dragQuad({ x: -0.2, y: 0.9 }, { x: 0.3, y: 1.4 });
    expect(q[0]).toEqual({ x: 0, y: 0.9 });
    expect(q[2]).toEqual({ x: 0.3, y: 1 });
  });
});

describe('usableBox', () => {
  it('drops a click that drew no area', () => {
    expect(usableBox(dragQuad({ x: 0.5, y: 0.5 }, { x: 0.501, y: 0.6 }))).toBe(false);
    expect(usableBox(dragQuad({ x: 0.5, y: 0.5 }, { x: 0.52, y: 0.52 }))).toBe(true);
  });
});

describe('quadPoints', () => {
  it('scales corners to the image size for an SVG polygon', () => {
    const q = dragQuad({ x: 0, y: 0 }, { x: 0.5, y: 0.25 });
    expect(quadPoints(q, 200, 400)).toBe('0,0 100,0 100,100 0,100');
  });
});

describe('boxesKey', () => {
  it('changes when a box moves', () => {
    const a = [dragQuad({ x: 0.1, y: 0.1 }, { x: 0.2, y: 0.2 })];
    const b = [dragQuad({ x: 0.1, y: 0.1 }, { x: 0.2, y: 0.3 })];
    expect(boxesKey(a)).toBe(boxesKey(structuredClone(a)));
    expect(boxesKey(a)).not.toBe(boxesKey(b));
  });
});
