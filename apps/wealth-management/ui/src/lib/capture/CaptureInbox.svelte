<script lang="ts">
  import type { CaptureLimits } from '$lib/api/types';
  import Icon from '$lib/components/Icon.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import {
    batchUsage,
    formatBytes,
    isImageFile,
    nextMaterialId,
    prepareImage,
    releaseItem,
    textBytes,
    type InboxItem,
  } from './materials';

  // Capture inbox (WM-CAP-1): drop, paste, pick or type materials.
  interface Props {
    items: InboxItem[];
    limits: CaptureLimits;
    disabled?: boolean;
    /** Images still being decoded and downscaled. */
    preparing?: number;
  }

  let { items = $bindable(), limits, disabled = false, preparing = $bindable(0) }: Props = $props();

  let text = $state('');
  let dragging = $state(false);
  let picker: HTMLInputElement;

  const usage = $derived(batchUsage(items, limits));

  async function addFiles(files: Iterable<File>) {
    for (const file of files) {
      if (!isImageFile(file)) {
        toasts.push('warning', `${file.name} is not an image. Paste text materials into the text box.`);
        continue;
      }
      preparing++;
      try {
        const { blob, mediaType } = await prepareImage(file, limits.maxItemBytes);
        const id = nextMaterialId(items);
        const name = file.name || `pasted-${id}.${mediaType.split('/')[1]}`;
        items = [
          ...items,
          { id, kind: 'image', name, size: blob.size, mediaType, blob, previewUrl: URL.createObjectURL(blob) },
        ];
      } catch (error) {
        toasts.error(error);
      } finally {
        preparing--;
      }
    }
  }

  function addText(value: string) {
    const trimmed = value.trim();
    if (!trimmed) return;
    const id = nextMaterialId(items);
    items = [...items, { id, kind: 'text', name: `Text ${id.slice(1)}`, size: textBytes(trimmed), text: trimmed }];
  }

  function remove(item: InboxItem) {
    releaseItem(item);
    items = items.filter((candidate) => candidate.id !== item.id);
  }

  // Paste anywhere on the page: images become materials, text goes to the box
  // unless the user is typing in another field.
  function onpaste(event: ClipboardEvent) {
    if (disabled || !event.clipboardData) return;
    const files = [...event.clipboardData.files].filter(isImageFile);
    if (files.length) {
      event.preventDefault();
      void addFiles(files);
      return;
    }
    const target = event.target as HTMLElement | null;
    const typing = target?.closest('input, textarea, select, [contenteditable="true"]');
    const pasted = event.clipboardData.getData('text/plain');
    if (!typing && pasted.trim()) {
      event.preventDefault();
      addText(pasted);
    }
  }

  function ondrop(event: DragEvent) {
    event.preventDefault();
    dragging = false;
    if (disabled || !event.dataTransfer) return;
    if (event.dataTransfer.files.length) void addFiles(event.dataTransfer.files);
    else addText(event.dataTransfer.getData('text/plain'));
  }
</script>

<svelte:window {onpaste} />

<div
  role="region"
  aria-label="Materials drop zone"
  class="rounded-xl border-2 border-dashed p-5 text-center transition {dragging
    ? 'border-indigo-500 bg-indigo-500/5'
    : 'border-slate-900/15 dark:border-white/15'}"
  ondragover={(event) => {
    event.preventDefault();
    dragging = !disabled;
  }}
  ondragleave={() => (dragging = false)}
  {ondrop}
>
  <Icon name="camera" size={28} class="muted mx-auto" />
  <p class="mt-2 text-sm font-medium">Drop screenshots here, paste them, or pick from your photos</p>
  <p class="muted mt-1 text-xs">
    Bank and wallet app screenshots, statements, notification or SMS text. PNG, JPEG, WebP, GIF and HEIC (Safari).
  </p>
  <button type="button" class="btn-ghost mt-3" {disabled} onclick={() => picker.click()}>
    <Icon name="plus" size={16} /> Choose images
  </button>
  <input
    bind:this={picker}
    type="file"
    accept="image/*,.heic,.heif"
    multiple
    class="hidden"
    onchange={(event) => {
      const input = event.currentTarget;
      if (input.files) void addFiles([...input.files]);
      input.value = '';
    }}
  />
</div>

<div class="mt-4">
  <label class="label" for="capture-text">Text material</label>
  <textarea
    id="capture-text"
    class="field min-h-24 font-mono text-xs"
    placeholder="Paste a notification, SMS or statement lines…"
    bind:value={text}
    {disabled}
  ></textarea>
  <div class="mt-2 flex justify-end">
    <button
      type="button"
      class="btn-ghost"
      disabled={disabled || !text.trim()}
      onclick={() => {
        addText(text);
        text = '';
      }}><Icon name="plus" size={16} /> Add text</button
    >
  </div>
</div>

{#if items.length || preparing}
  <ul class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5" aria-label="Materials">
    {#each items as item (item.id)}
      <li class="glass relative overflow-hidden p-2">
        {#if item.kind === 'image'}
          <img src={item.previewUrl} alt={item.name} class="h-28 w-full rounded-md object-cover object-top" />
        {:else}
          <pre
            class="h-28 overflow-hidden rounded-md bg-slate-900/5 p-2 text-[10px] whitespace-pre-wrap dark:bg-white/5">{item.text}</pre>
        {/if}
        <p class="mt-1 truncate text-xs font-medium" title={item.name}>{item.id} · {item.name}</p>
        <p class="muted text-xs">{formatBytes(item.size)}</p>
        <button
          type="button"
          class="glass absolute top-1 right-1 grid size-7 place-items-center"
          aria-label="Remove {item.name}"
          {disabled}
          onclick={() => remove(item)}><Icon name="x" size={14} /></button
        >
      </li>
    {/each}
    {#if preparing}
      <li class="glass muted grid h-40 place-items-center p-2 text-xs">Preparing {preparing} image(s)…</li>
    {/if}
  </ul>
{/if}

<p class="muted mt-3 text-xs">
  {usage.count} / {limits.maxItems} materials · {formatBytes(usage.bytes)} / {formatBytes(limits.maxBatchBytes)} · up to
  {formatBytes(limits.maxItemBytes)} each
</p>
{#each usage.problems as problem (problem)}
  <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">{problem}</p>
{/each}
