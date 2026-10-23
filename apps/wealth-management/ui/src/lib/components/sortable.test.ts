import { afterEach, describe, expect, it, vi } from 'vitest';
import { indexAt, sortable } from './sortable';

const rect = (left: number, top: number, width = 100, height = 100) => ({
  left,
  top,
  right: left + width,
  bottom: top + height,
});

describe('indexAt', () => {
  const rects = [rect(0, 0), rect(110, 0), rect(0, 110)];

  it('returns the rect under the point', () => {
    expect(indexAt(rects, 50, 50)).toBe(0);
    expect(indexAt(rects, 150, 20)).toBe(1);
    expect(indexAt(rects, 10, 200)).toBe(2);
  });

  it('returns -1 in a gap or outside', () => {
    expect(indexAt(rects, 105, 50)).toBe(-1);
    expect(indexAt(rects, 500, 500)).toBe(-1);
  });
});

// A 2 x 2 grid of 100px cards with a 10px gap, each with a drag handle.
function grid(count = 4) {
  const list = document.createElement('ul');
  for (let index = 0; index < count; index++) {
    const item = document.createElement('li');
    const handle = document.createElement('span');
    handle.dataset.dragHandle = '';
    item.append(handle);
    const box = rect((index % 2) * 110, Math.floor(index / 2) * 110);
    item.getBoundingClientRect = () => ({ ...box, x: box.left, y: box.top, width: 100, height: 100 }) as DOMRect;
    list.append(item);
  }
  document.body.append(list);
  return { list, items: [...list.children] as HTMLElement[] };
}

const pointer = (target: EventTarget, type: string, x: number, y: number) =>
  target.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, button: 0 }));

describe('sortable', () => {
  afterEach(() => (document.body.innerHTML = ''));

  it('Given a card dragged by its handle When dropped on another card Then it reports the move and clears the drag marks', () => {
    const onmove = vi.fn();
    const { list, items } = grid();
    const cleanup = sortable({ onmove })(list);

    pointer(items[0].querySelector('[data-drag-handle]')!, 'pointerdown', 50, 50);
    expect(items[0].dataset.dragging).toBe('');
    pointer(window, 'pointermove', 160, 160);
    expect(items[3].dataset.dropTarget).toBe('');
    expect(items[0].style.transform).toBe('translate(110px, 110px)');
    pointer(window, 'pointerup', 160, 160);

    expect(onmove).toHaveBeenCalledWith(0, 3);
    expect(items[0].dataset.dragging).toBeUndefined();
    expect(items[3].dataset.dropTarget).toBeUndefined();
    expect(items[0].style.transform).toBe('');
    cleanup?.();
  });

  it('Given a drag When dropped back on its own place, or cancelled with Escape Then nothing moves', () => {
    const onmove = vi.fn();
    const { list, items } = grid();
    sortable({ onmove })(list);
    const handle = items[1].querySelector('[data-drag-handle]')!;

    pointer(handle, 'pointerdown', 150, 50);
    pointer(window, 'pointermove', 50, 50);
    pointer(window, 'pointermove', 160, 40);
    pointer(window, 'pointerup', 160, 40);

    pointer(handle, 'pointerdown', 150, 50);
    pointer(window, 'pointermove', 50, 50);
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    pointer(window, 'pointerup', 50, 50);

    expect(onmove).not.toHaveBeenCalled();
    expect(items[0].dataset.dropTarget).toBeUndefined();
  });

  it('Given a press outside the handle, or a disabled list When dragged Then nothing starts', () => {
    const onmove = vi.fn();
    const { list, items } = grid();
    sortable({ onmove })(list);
    pointer(items[0], 'pointerdown', 50, 50);
    pointer(window, 'pointermove', 160, 160);
    pointer(window, 'pointerup', 160, 160);

    const other = grid();
    sortable({ onmove, disabled: true })(other.list);
    pointer(other.items[0].querySelector('[data-drag-handle]')!, 'pointerdown', 50, 50);
    pointer(window, 'pointermove', 160, 160);
    pointer(window, 'pointerup', 160, 160);

    expect(onmove).not.toHaveBeenCalled();
    expect(items[0].dataset.dragging).toBeUndefined();
  });
});
