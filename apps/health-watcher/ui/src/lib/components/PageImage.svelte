<script lang="ts">
  import { pageImageURL } from '$lib/api/client';
  import type { Page, Point, Quad } from '$lib/api/types';
  import { boxesKey, dragQuad, quadPoints, usableBox } from '$lib/boxes';
  import { session } from '$lib/state/session.svelte';

  // One page's rendered image, or the masked copy the AI provider saw
  // (HW-FIND-3, HW-AI-7). With `drawing`, dragging on the page adds a black
  // box and clicking a box removes it; `onboxes` gets the new list.
  interface Props {
    page: Page;
    masked?: boolean;
    zoom?: number;
    drawing?: boolean;
    boxes?: Quad[];
    onboxes?: (boxes: Quad[]) => void;
  }

  let { page, masked = false, zoom = 1, drawing = false, boxes = [], onboxes }: Props = $props();

  // A new mask has a new image; a hash of its boxes makes the browser fetch it.
  const version = $derived(boxesKey(page.mask?.boxes ?? []));
  const src = $derived(
    session.tenantId ? `${pageImageURL(session.tenantId, page.id, masked)}${masked ? `&v=${version}` : ''}` : '',
  );
  const w = $derived(page.width || 1000);
  const h = $derived(page.height || 1414);

  let svg: SVGSVGElement | undefined = $state();
  let start = $state<Point | null>(null);
  let current = $state<Point | null>(null);
  const preview = $derived(start && current ? dragQuad(start, current) : null);

  function at(event: PointerEvent): Point {
    const r = svg!.getBoundingClientRect();
    return { x: (event.clientX - r.left) / r.width, y: (event.clientY - r.top) / r.height };
  }

  function down(event: PointerEvent) {
    if (!drawing || event.button !== 0) return;
    svg!.setPointerCapture(event.pointerId);
    start = current = at(event);
  }

  function up() {
    if (preview && usableBox(preview)) onboxes?.([...boxes, preview]);
    start = current = null;
  }
</script>

<div class="overflow-auto rounded-xl border border-slate-900/10 bg-white dark:border-white/10">
  {#if page.hasImage}
    <div class="relative" style="width: {zoom * 100}%">
      <img
        {src}
        alt="Page {page.number}{masked ? ', masked' : ''}"
        class="block w-full select-none"
        draggable="false"
      />
      {#if drawing}
        <svg
          bind:this={svg}
          class="absolute inset-0 size-full cursor-crosshair touch-none"
          viewBox="0 0 {w} {h}"
          preserveAspectRatio="none"
          role="application"
          aria-label="Drag to black out an area; click a box to remove it"
          onpointerdown={down}
          onpointermove={(e) => {
            if (start) current = at(e);
          }}
          onpointerup={up}
          onpointercancel={() => (start = current = null)}
        >
          {#each boxes as box, i (i)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <polygon
              points={quadPoints(box, w, h)}
              class="cursor-pointer fill-rose-500/30 stroke-rose-600"
              stroke-width={Math.max(2, w / 400)}
              role="button"
              tabindex="-1"
              aria-label="Remove this box"
              onpointerdown={(e) => e.stopPropagation()}
              onclick={() => onboxes?.(boxes.filter((_, j) => j !== i))}
            />
          {/each}
          {#if preview}
            <polygon
              points={quadPoints(preview, w, h)}
              class="fill-slate-900/40 stroke-slate-900"
              stroke-width={Math.max(2, w / 400)}
            />
          {/if}
        </svg>
      {/if}
    </div>
  {:else}
    <p class="muted p-6 text-center text-sm">
      No image for this page: the installed mac-ocr can't render pages yet. Rebuild it to see and send page images.
    </p>
  {/if}
</div>
