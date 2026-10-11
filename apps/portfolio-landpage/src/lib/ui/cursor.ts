export type BubblePlacement = { horizontal: 'left' | 'right'; vertical: 'above' | 'below' };

type Point = { x: number; y: number };
type Size = { width: number; height: number };

/** Width and height of the expanded cursor bubble, plus the gap between its tail and the pointer. InkCursor.svelte reads both as CSS variables. */
export const BUBBLE_SIZE = 64;
export const BUBBLE_GAP = 6;
const ROOM_NEEDED = BUBBLE_SIZE + BUBBLE_GAP * 2;

/**
 * Where the speech bubble sits relative to the pointer. It prefers above-right, clear of the native cursor, which
 * hangs down and to the right of its hotspot, and flips on whichever axis would push it off-screen.
 */
export const bubblePlacement = (pointer: Point, viewport: Size): BubblePlacement => ({
  horizontal: pointer.x + ROOM_NEEDED > viewport.width ? 'left' : 'right',
  vertical: pointer.y < ROOM_NEEDED ? 'below' : 'above',
});
