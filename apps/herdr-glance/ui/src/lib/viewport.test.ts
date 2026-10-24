import { describe, expect, it, vi } from 'vitest';
import { fitVisualViewport } from './viewport';

/** A stand-in for iOS Safari's visual viewport and window scroll. */
function fakeWindow(height: number) {
  const viewport = Object.assign(new EventTarget(), { height, offsetTop: 0 });
  const win = { visualViewport: viewport, innerHeight: height, scrollY: 0, scrollTo: vi.fn() };
  const move = (next: { height: number; offsetTop: number; scrollY: number }) => {
    viewport.height = next.height;
    viewport.offsetTop = next.offsetTop;
    win.scrollY = next.scrollY;
    viewport.dispatchEvent(new Event('resize'));
  };
  return { win: win as unknown as Window, scrollTo: win.scrollTo, move, viewport };
}

describe('fitVisualViewport', () => {
  it('sizes the page to the visible area, above the keyboard', () => {
    const { win, move } = fakeWindow(844);
    const node = document.createElement('div');
    fitVisualViewport(node, win);
    expect(node.style.height).toBe('844px');

    // The keyboard opens and Safari scrolls the page up by 120px.
    move({ height: 508, offsetTop: 120, scrollY: 120 });
    expect(node.style.height).toBe('508px');
    expect(node.style.transform).toBe('translateY(120px)');
  });

  it('scrolls back to the top when the keyboard closes, so the header shows again', () => {
    const { win, move, scrollTo } = fakeWindow(844);
    const node = document.createElement('div');
    fitVisualViewport(node, win);
    move({ height: 508, offsetTop: 120, scrollY: 120 });
    expect(scrollTo).not.toHaveBeenCalled();

    // The keyboard closes, but iOS leaves the page scrolled.
    move({ height: 844, offsetTop: 0, scrollY: 120 });
    expect(scrollTo).toHaveBeenCalledWith(0, 0);
    expect(node.style.height).toBe('844px');
    expect(node.style.transform).toBe('');
  });

  it('stops listening when the page goes away', () => {
    const { win, move } = fakeWindow(844);
    const node = document.createElement('div');
    const stop = fitVisualViewport(node, win);
    stop();
    move({ height: 508, offsetTop: 120, scrollY: 120 });
    expect(node.style.height).toBe('844px');
  });

  it('does nothing without a visual viewport', () => {
    const node = document.createElement('div');
    fitVisualViewport(node, { visualViewport: null } as unknown as Window)();
    expect(node.style.height).toBe('');
  });
});
