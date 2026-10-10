<!--
  Adaptive pointer: an ink blob trails the mouse and swells into a labelled bubble
  over interactive elements. Labels come from data-cursor, falling back to the
  element type. Decorative only; the native cursor is kept for accessibility.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { prefersReducedMotion } from 'svelte/motion';

  const INTERACTIVE = 'a, button, [data-cursor], summary, label';

  let isEnabled = $state(false);
  let isVisible = $state(false);
  let label = $state('');
  let x = $state(0);
  let y = $state(0);

  const labelFor = (element: Element): string => {
    const custom = element.getAttribute('data-cursor');
    if (custom) return custom;
    if (element instanceof HTMLAnchorElement) {
      return element.target === '_blank' || /^https?:/.test(element.getAttribute('href') ?? '') ? 'go ↗' : 'open';
    }
    return 'tap';
  };

  onMount(() => {
    const finePointer = window.matchMedia('(pointer: fine)');
    if (!finePointer.matches || prefersReducedMotion.current) return;
    isEnabled = true;

    let targetX = 0;
    let targetY = 0;
    let frame = 0;

    const tick = () => {
      x += (targetX - x) * 0.22;
      y += (targetY - y) * 0.22;
      frame = requestAnimationFrame(tick);
    };

    const onMove = (event: PointerEvent) => {
      targetX = event.clientX;
      targetY = event.clientY;
      if (!isVisible) {
        x = targetX;
        y = targetY;
        isVisible = true;
      }
    };

    const onOver = (event: PointerEvent) => {
      const target = (event.target as Element | null)?.closest(INTERACTIVE);
      label = target ? labelFor(target) : '';
    };

    const onLeave = () => (isVisible = false);

    window.addEventListener('pointermove', onMove, { passive: true });
    window.addEventListener('pointerover', onOver, { passive: true });
    document.documentElement.addEventListener('pointerleave', onLeave);
    frame = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerover', onOver);
      document.documentElement.removeEventListener('pointerleave', onLeave);
    };
  });
</script>

{#if isEnabled}
  <div
    class="cursor pointer-events-none fixed top-0 left-0 z-[90]"
    class:active={label !== ''}
    class:hidden-cursor={!isVisible}
    style:transform="translate3d({x}px, {y}px, 0)"
    aria-hidden="true"
  >
    <div
      class="blob grid place-items-center border-[3px] border-rule bg-accent font-mono text-[11px] font-bold text-accent-ink uppercase"
    >
      <span>{label}</span>
    </div>
  </div>
{/if}

<style>
  .cursor {
    transition: opacity 200ms ease;
  }

  .hidden-cursor {
    opacity: 0;
  }

  .blob {
    width: 14px;
    height: 14px;
    border-radius: 9999px;
    transform: translate(-50%, -50%);
    transition:
      width 220ms cubic-bezier(0.3, 1.6, 0.5, 1),
      height 220ms cubic-bezier(0.3, 1.6, 0.5, 1),
      border-radius 220ms ease;
    overflow: hidden;
    white-space: nowrap;
  }

  .blob span {
    opacity: 0;
    transition: opacity 120ms ease;
  }

  .active .blob {
    width: 64px;
    height: 64px;
    transform: translate(10%, 10%);
    border-radius: 50% 50% 50% 8px;
  }

  .active .blob span {
    opacity: 1;
  }
</style>
