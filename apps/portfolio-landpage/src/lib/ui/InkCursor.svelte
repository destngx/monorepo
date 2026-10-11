<!--
  Adaptive pointer: an ink blob trails the mouse and swells into a labelled bubble
  over interactive elements. Labels come from data-cursor, falling back to the
  element type. Decorative only; the native cursor is kept for accessibility.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { prefersReducedMotion } from 'svelte/motion';
  import { BUBBLE_GAP, BUBBLE_SIZE, bubblePlacement, type BubblePlacement } from './cursor';

  const INTERACTIVE = 'a, button, [data-cursor], summary, label';

  let isEnabled = $state(false);
  let isVisible = $state(false);
  let label = $state('');
  let x = $state(0);
  let y = $state(0);
  // Kept as two primitives so a pointer move that lands on the same placement does not re-render.
  let horizontal = $state<BubblePlacement['horizontal']>('right');
  let vertical = $state<BubblePlacement['vertical']>('above');

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
      ({ horizontal, vertical } = bubblePlacement(
        { x: targetX, y: targetY },
        { width: window.innerWidth, height: window.innerHeight },
      ));
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
    data-horizontal={horizontal}
    data-vertical={vertical}
    style:--bubble-size="{BUBBLE_SIZE}px"
    style:--bubble-gap="{BUBBLE_GAP}px"
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

  /* The bubble grows out of the pointer: its tail (the one sharp corner) always sits next to the hotspot. */
  .active .blob {
    width: var(--bubble-size);
    height: var(--bubble-size);
    transform: translate(var(--shift-x), var(--shift-y));
    border-radius: 50%;
  }

  .cursor[data-horizontal='right'] {
    --shift-x: var(--bubble-gap);
  }

  .cursor[data-horizontal='left'] {
    --shift-x: calc(-100% - var(--bubble-gap));
  }

  .cursor[data-vertical='above'] {
    --shift-y: calc(-100% - var(--bubble-gap));
  }

  .cursor[data-vertical='below'] {
    --shift-y: var(--bubble-gap);
  }

  .active[data-horizontal='right'][data-vertical='above'] .blob {
    border-bottom-left-radius: 8px;
  }

  .active[data-horizontal='right'][data-vertical='below'] .blob {
    border-top-left-radius: 8px;
  }

  .active[data-horizontal='left'][data-vertical='above'] .blob {
    border-bottom-right-radius: 8px;
  }

  .active[data-horizontal='left'][data-vertical='below'] .blob {
    border-top-right-radius: 8px;
  }

  .active .blob span {
    opacity: 1;
  }
</style>
