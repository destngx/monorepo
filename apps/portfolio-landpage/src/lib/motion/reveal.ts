import type { Action } from 'svelte/action';

export type RevealOptions = { delay?: number; threshold?: number };

/**
 * Fades/slides an element in the first time it scrolls into view.
 * Styling lives in app.css under [data-reveal]; reduced motion is handled there too.
 */
export const reveal: Action<HTMLElement, RevealOptions | undefined> = (node, options = {}) => {
  const { delay = 0, threshold = 0.15 } = options;
  node.style.setProperty('--reveal-delay', `${delay}ms`);

  if (typeof IntersectionObserver === 'undefined') {
    node.dataset.reveal = 'shown';
    return;
  }

  node.dataset.reveal = 'hidden';
  const observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        node.dataset.reveal = 'shown';
        observer.disconnect();
      }
    },
    { threshold, rootMargin: '0px 0px -10% 0px' },
  );
  observer.observe(node);

  return {
    destroy() {
      observer.disconnect();
    },
  };
};
