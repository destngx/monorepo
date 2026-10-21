<script lang="ts">
  import { fileURL } from '$lib/api/client';
  import type { StoredFile } from '$lib/api/types';
  import { formatSize, isImage } from '$lib/format';
  import { session } from '$lib/state/session.svelte';
  import Icon from './Icon.svelte';

  // One stored file: an image shown inline, a PDF in the browser's viewer,
  // anything else (HEIC) as a download (HW-FIND-3).
  let { file, compact = false }: { file: StoredFile; compact?: boolean } = $props();
  const src = $derived(session.tenantId ? fileURL(session.tenantId, file.id) : '');
  const download = $derived(session.tenantId ? fileURL(session.tenantId, file.id, true) : '');
  const name = $derived(file.originalName || 'file');
</script>

{#if compact}
  <a
    href={src}
    target="_blank"
    rel="noopener"
    class="group relative block aspect-[3/4] overflow-hidden rounded-xl border border-slate-900/10 bg-white/70 dark:border-white/10 dark:bg-slate-900/50"
    title={name}
  >
    {#if isImage(file.mime)}
      <img {src} alt={name} loading="lazy" class="size-full object-cover" />
    {:else}
      <span class="grid size-full place-items-center p-2 text-center">
        <span class="flex flex-col items-center gap-1">
          <Icon name={file.mime === 'application/pdf' ? 'file' : 'image'} size={28} class="text-teal-600" />
          <span class="muted line-clamp-2 text-[11px] break-all">{name}</span>
        </span>
      </span>
    {/if}
  </a>
{:else}
  <figure
    class="overflow-hidden rounded-xl border border-slate-900/10 bg-white/70 dark:border-white/10 dark:bg-slate-900/50"
  >
    {#if isImage(file.mime)}
      <img {src} alt={name} loading="lazy" class="max-h-[80vh] w-full object-contain" />
    {:else if file.mime === 'application/pdf'}
      <iframe {src} title={name} class="h-[75vh] w-full"></iframe>
    {:else}
      <div class="muted p-6 text-center text-sm">This browser can't show {file.mime} files. Download it to view.</div>
    {/if}
    <figcaption
      class="flex items-center justify-between gap-2 border-t border-slate-900/10 px-3 py-2 text-xs dark:border-white/10"
    >
      <span class="truncate">{name} · {formatSize(file.size)}</span>
      <span class="flex shrink-0 gap-3">
        <a class="link" href={src} target="_blank" rel="noopener">Open</a>
        <a class="link" href={download}>Download</a>
      </span>
    </figcaption>
  </figure>
{/if}
