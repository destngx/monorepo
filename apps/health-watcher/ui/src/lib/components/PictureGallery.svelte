<script lang="ts">
  import { pictureImageURL } from '$lib/api/client';
  import type { Picture } from '$lib/api/types';
  import { pictureKindLabels } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  // The pictures cut out of a document's pages, such as X-ray films and
  // ECG traces (HW-READ-11). Each opens at full size in a new tab and names
  // the page it came from. They are shown, never read with AI (D5).
  interface Props {
    pictures: Picture[];
    pageOf: (id: string | undefined) => { number: number; name: string } | undefined;
    pageHref?: (pageId: string) => string;
  }
  let { pictures, pageOf, pageHref }: Props = $props();

  const src = (p: Picture) => (session.tenantId ? pictureImageURL(session.tenantId, p.id) : '');
</script>

<ul class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
  {#each pictures as p (p.id)}
    {@const page = pageOf(p.pageId)}
    <li class="panel flex min-w-0 flex-col gap-2 p-3">
      <a href={src(p)} target="_blank" rel="noopener" class="block bg-ink-900/5 dark:bg-ink-50/5">
        <img
          src={src(p)}
          alt="{pictureKindLabels[p.kind]}{p.caption ? `: ${p.caption}` : ''}"
          width={p.width}
          height={p.height}
          loading="lazy"
          class="mx-auto max-h-72 w-auto object-contain"
        />
      </a>
      <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm">
        <span class="tag shrink-0">{pictureKindLabels[p.kind]}</span>
        {#if page}
          {#if pageHref}
            <a class="link min-w-0 truncate text-xs" href={pageHref(p.pageId)}>{page.name}</a>
          {:else}
            <span class="muted min-w-0 truncate text-xs">{page.name}</span>
          {/if}
        {/if}
      </div>
      {#if p.caption}
        <p class="text-sm whitespace-pre-line">{p.caption}</p>
      {/if}
    </li>
  {/each}
</ul>
