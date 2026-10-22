<script lang="ts">
  import { pageImageURL } from '$lib/api/client';
  import type { MaskBox, Page, Point, Quad } from '$lib/api/types';
  import { boxesKey, dragQuad, labelAt, quadPoints, usableBox } from '$lib/boxes';
  import { session } from '$lib/state/session.svelte';

  // One page's rendered image, or the masked copy the AI provider saw
  // (HW-FIND-3, HW-AI-7). With `drawing`, dragging on the page adds a black
  // box and clicking a box removes it; `onboxes` gets the new list. On the
  // masked copy every box shows its number: the rule boxes (`auto`) first,
  // then the drawn ones. With `ontoggle`, clicking a rule box turns it off
  // or back on.
  interface Props {
    page: Page;
    masked?: boolean;
    zoom?: number;
    drawing?: boolean;
    boxes?: Quad[];
    onboxes?: (boxes: Quad[]) => void;
    auto?: MaskBox[];
    ontoggle?: (key: string) => void;
  }

  let { page, masked = false, zoom = 1, drawing = false, boxes = [], onboxes, auto = [], ontoggle }: Props = $props();

  // A new mask has a new image; a hash of its boxes makes the browser fetch it.
  const version = $derived(boxesKey(page.mask?.boxes ?? []));
  const src = $derived(
    session.tenantId ? `${pageImageURL(session.tenantId, page.id, masked)}${masked ? `&v=${version}` : ''}` : '',
  );
  const w = $derived(page.width || 1000);
  const h = $derived(page.height || 1414);
  const overlay = $derived(drawing || (masked && auto.length + boxes.length > 0));
  const stroke = $derived(Math.max(2, w / 400));
  const font = $derived(Math.max(14, w / 60));

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

{#snippet number(n: number, q: Quad, fill: string)}
  {@const at = labelAt(q, w, h)}
  {@const width = font * (0.35 + 0.6 * String(n).length)}
  <g class="pointer-events-none" transform="translate({at.x}, {Math.max(0, at.y - font * 1.15)})">
    <rect {width} height={font * 1.15} rx={font * 0.25} class={fill} />
    <text x={width / 2} y={font * 0.85} font-size={font} text-anchor="middle" class="fill-white font-semibold">{n}</text
    >
  </g>
{/snippet}

<div class="overflow-auto rounded-xl border border-slate-900/10 bg-white dark:border-white/10">
  {#if page.hasImage}
    <div class="relative" style="width: {zoom * 100}%">
      <img
        {src}
        alt="Page {page.number}{masked ? ', masked' : ''}"
        class="block w-full select-none"
        draggable="false"
      />
      {#if overlay}
        <svg
          bind:this={svg}
          class="absolute inset-0 size-full touch-none {drawing ? 'cursor-crosshair' : 'pointer-events-none'}"
          viewBox="0 0 {w} {h}"
          preserveAspectRatio="none"
          role="application"
          aria-label={drawing ? 'Drag to black out an area; click a box to remove it' : 'Numbered black boxes'}
          onpointerdown={down}
          onpointermove={(e) => {
            if (start) current = at(e);
          }}
          onpointerup={up}
          onpointercancel={() => (start = current = null)}
        >
          {#each auto as box, i (box.key)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <polygon
              points={quadPoints(box.quad, w, h)}
              class="{box.off ? 'fill-amber-400/20 stroke-amber-500' : 'fill-transparent stroke-teal-500'} {ontoggle
                ? 'pointer-events-auto cursor-pointer'
                : 'pointer-events-none'}"
              stroke-width={stroke}
              stroke-dasharray={box.off ? `${stroke * 3} ${stroke * 2}` : undefined}
              role="button"
              tabindex="-1"
              aria-label="Box {i + 1}: {box.off ? 'sent to the AI, click to hide' : 'hidden, click to send'}"
              onpointerdown={(e) => e.stopPropagation()}
              onclick={() => ontoggle?.(box.key)}
            />
            {@render number(i + 1, box.quad, box.off ? 'fill-amber-500' : 'fill-teal-600')}
          {/each}
          {#each boxes as box, i (i)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <polygon
              points={quadPoints(box, w, h)}
              class="fill-rose-500/30 stroke-rose-600 {drawing ? 'cursor-pointer' : 'pointer-events-none'}"
              stroke-width={stroke}
              role="button"
              tabindex="-1"
              aria-label="Remove this box"
              onpointerdown={(e) => e.stopPropagation()}
              onclick={() => drawing && onboxes?.(boxes.filter((_, j) => j !== i))}
            />
            {@render number(auto.length + i + 1, box, 'fill-rose-600')}
          {/each}
          {#if preview}
            <polygon
              points={quadPoints(preview, w, h)}
              class="fill-slate-900/40 stroke-slate-900"
              stroke-width={stroke}
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
