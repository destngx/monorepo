function noop(): void {
  // Nothing to stop: the browser has no visual viewport.
}

/**
 * Keeps a full-screen page inside the part of the screen the keyboard leaves
 * free. iOS Safari does not shrink the page for the keyboard: it scrolls the
 * whole page up instead, and often leaves it scrolled after the keyboard
 * closes, which hides the header. This sizes and moves the page to the
 * visual viewport, and scrolls back to the top once the keyboard is gone.
 * Returns the function that stops it.
 */
export function fitVisualViewport(node: HTMLElement, win: Window = window): () => void {
  const viewport = win.visualViewport;
  if (!viewport) return noop;

  const update = () => {
    node.style.height = `${viewport.height}px`;
    node.style.transform = viewport.offsetTop > 0 ? `translateY(${viewport.offsetTop}px)` : '';
    // The keyboard is closed when the visible area is the full window again.
    if (viewport.height >= win.innerHeight - 1 && win.scrollY !== 0) win.scrollTo(0, 0);
  };
  update();
  viewport.addEventListener('resize', update);
  viewport.addEventListener('scroll', update);
  return () => {
    viewport.removeEventListener('resize', update);
    viewport.removeEventListener('scroll', update);
  };
}
