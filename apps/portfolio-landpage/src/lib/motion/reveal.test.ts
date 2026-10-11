import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { reveal } from './reveal';

type Callback = (entries: Array<Partial<IntersectionObserverEntry>>) => void;

describe('reveal action', () => {
  let callback: Callback;
  const observe = vi.fn();
  const disconnect = vi.fn();

  beforeEach(() => {
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn(function (cb: Callback) {
        callback = cb;
        return { observe, disconnect, unobserve: vi.fn() };
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('marks the node hidden, then revealed once it intersects', () => {
    const node = document.createElement('div');
    reveal(node);

    expect(node.dataset.reveal).toBe('hidden');
    expect(observe).toHaveBeenCalledWith(node);

    callback([{ isIntersecting: true, target: node }]);

    expect(node.dataset.reveal).toBe('shown');
    expect(disconnect).toHaveBeenCalled();
  });

  it('applies a stagger delay', () => {
    const node = document.createElement('div');
    reveal(node, { delay: 120 });

    expect(node.style.getPropertyValue('--reveal-delay')).toBe('120ms');
  });

  it('shows content immediately when IntersectionObserver is unavailable', () => {
    vi.stubGlobal('IntersectionObserver', undefined);
    const node = document.createElement('div');
    reveal(node);

    expect(node.dataset.reveal).toBe('shown');
  });

  it('disconnects on destroy', () => {
    const node = document.createElement('div');
    reveal(node)?.destroy?.();

    expect(disconnect).toHaveBeenCalled();
  });
});
