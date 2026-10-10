import { describe, expect, it } from 'vitest';
import { bubblePlacement } from './cursor';

const viewport = { width: 1280, height: 800 };

describe('bubblePlacement', () => {
  it('floats the bubble above and to the right of the pointer by default', () => {
    expect(bubblePlacement({ x: 400, y: 400 }, viewport)).toEqual({ horizontal: 'right', vertical: 'above' });
  });

  it('drops below the pointer when there is no room above, such as on the header nav', () => {
    expect(bubblePlacement({ x: 400, y: 40 }, viewport)).toEqual({ horizontal: 'right', vertical: 'below' });
  });

  it('flips to the left of the pointer near the right edge', () => {
    expect(bubblePlacement({ x: 1250, y: 400 }, viewport)).toEqual({ horizontal: 'left', vertical: 'above' });
  });

  it('flips both ways in the top-right corner', () => {
    expect(bubblePlacement({ x: 1250, y: 20 }, viewport)).toEqual({ horizontal: 'left', vertical: 'below' });
  });
});
