import { describe, expect, it } from 'vitest';
import type { MaskBox } from './api/types';
import { boxesKey, dragQuad, labelAt, quadPoints, toggleOff, usableBox } from './boxes';

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

describe('labelAt', () => {
  it('puts the number at the top-left of the box bounds', () => {
    const q = dragQuad({ x: 0.5, y: 0.25 }, { x: 0.25, y: 0.75 });
    expect(labelAt(q, 200, 400)).toEqual({ x: 50, y: 100 });
  });
});

describe('toggleOff', () => {
  const box = (key: string, off: boolean): MaskBox => ({
    key,
    kind: 'line',
    quad: dragQuad({ x: 0, y: 0 }, { x: 0.1, y: 0.1 }),
    text: '',
    off,
  });
  const auto = [box('line:1', false), box('line:4', true), box('barcode:0', false)];

  it('turns a box off and keeps the others as they are', () => {
    expect(toggleOff(auto, 'line:1')).toEqual(['line:1', 'line:4']);
  });
  it('turns an off box back on', () => {
    expect(toggleOff(auto, 'line:4')).toEqual([]);
  });
});
