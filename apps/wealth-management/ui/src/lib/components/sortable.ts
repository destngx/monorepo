import type { Attachment } from 'svelte/attachments';

type Box = Pick<DOMRect, 'left' | 'top' | 'right' | 'bottom'>;

export interface SortableOptions {
  /** Ignores new drags, for example while another change is saving. */
  disabled?: boolean;
  /** Called on drop when the item lands on another item's place. */
  onmove: (from: number, to: number) => void;
}

/** The index of the box under the point, or -1 in a gap or outside. */
export function indexAt(boxes: Box[], x: number, y: number): number {
  return boxes.findIndex((box) => x >= box.left && x <= box.right && y >= box.top && y <= box.bottom);
}

// Pointer distance from the top or bottom of the viewport that scrolls the page.
// Wider than the sticky header and the phone tab bar, so the zone shows past them.
const SCROLL_EDGE = 120;
const SCROLL_STEP = 12;

/**
 * Reorders the list's children by dragging an element marked `data-drag-handle`.
 * Pointer events, so mouse, pen and touch all work. The dragged child follows the
 * pointer and gets `data-dragging`; the child it would land on gets `data-drop-target`.
 * Escape cancels. Keyboard users need another way to reorder (move buttons).
 */
export function sortable(options: SortableOptions): Attachment<HTMLElement> {
  return (list) => {
    let drag: {
      item: HTMLElement;
      items: HTMLElement[];
      from: number;
      to: number;
      // Where the press started, in page coordinates, so scrolling keeps the item under the pointer.
      startX: number;
      startY: number;
      // The dragged item's own place, in page coordinates.
      origin: Box;
      x: number;
      y: number;
      frame: number;
    } | null = null;

    function place() {
      if (!drag) return;
      const { item, items, origin } = drag;
      item.style.transform = `translate(${drag.x + scrollX - drag.startX}px, ${drag.y + scrollY - drag.startY}px)`;
      const own = {
        left: origin.left - scrollX,
        right: origin.right - scrollX,
        top: origin.top - scrollY,
        bottom: origin.bottom - scrollY,
      };
      const index = indexAt(
        items.map((other) => (other === item ? own : other.getBoundingClientRect())),
        drag.x,
        drag.y,
      );
      if (index === -1 || index === drag.to) return;
      delete items[drag.to].dataset.dropTarget;
      drag.to = index;
      if (index !== drag.from) items[index].dataset.dropTarget = '';
    }

    function autoScroll() {
      if (!drag) return;
      const step = drag.y < SCROLL_EDGE ? -SCROLL_STEP : drag.y > innerHeight - SCROLL_EDGE ? SCROLL_STEP : 0;
      if (step) {
        scrollBy(0, step);
        place();
      }
      drag.frame = requestAnimationFrame(autoScroll);
    }

    function onmove(event: PointerEvent) {
      if (!drag) return;
      event.preventDefault();
      drag.x = event.clientX;
      drag.y = event.clientY;
      place();
    }

    function end(commit: boolean) {
      if (!drag) return;
      const { item, items, from, to, frame } = drag;
      drag = null;
      cancelAnimationFrame(frame);
      removeEventListener('pointermove', onmove);
      removeEventListener('pointerup', onup);
      removeEventListener('pointercancel', oncancel);
      removeEventListener('keydown', onkey);
      item.style.transform = '';
      delete item.dataset.dragging;
      delete items[to].dataset.dropTarget;
      if (commit && to !== from) options.onmove(from, to);
    }

    const onup = () => end(true);
    const oncancel = () => end(false);
    function onkey(event: KeyboardEvent) {
      if (event.key === 'Escape') end(false);
    }

    function ondown(event: PointerEvent) {
      if (drag || options.disabled || event.button !== 0) return;
      const handle = event.target instanceof Element ? event.target.closest('[data-drag-handle]') : null;
      const items = [...list.children].filter((child): child is HTMLElement => child instanceof HTMLElement);
      const from = handle ? items.findIndex((item) => item.contains(handle)) : -1;
      if (from === -1) return;
      // No text selection, no page scroll from the press.
      event.preventDefault();
      const item = items[from];
      const box = item.getBoundingClientRect();
      drag = {
        item,
        items,
        from,
        to: from,
        startX: event.clientX + scrollX,
        startY: event.clientY + scrollY,
        origin: {
          left: box.left + scrollX,
          right: box.right + scrollX,
          top: box.top + scrollY,
          bottom: box.bottom + scrollY,
        },
        x: event.clientX,
        y: event.clientY,
        frame: requestAnimationFrame(autoScroll),
      };
      item.dataset.dragging = '';
      addEventListener('pointermove', onmove, { passive: false });
      addEventListener('pointerup', onup);
      addEventListener('pointercancel', oncancel);
      addEventListener('keydown', onkey);
    }

    list.addEventListener('pointerdown', ondown);
    return () => {
      end(false);
      list.removeEventListener('pointerdown', ondown);
    };
  };
}
