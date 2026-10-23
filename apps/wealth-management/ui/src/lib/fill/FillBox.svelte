<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { FillFormKey, FillNote, FillResult, FillValues } from '$lib/api/types';
  import { isImageFile, prepareImage, toBase64 } from '$lib/capture/materials';
  import { Button, Icon } from '@libs/ui-kit';
  import { onDestroy } from 'svelte';
  import { FILL_EXAMPLES, FILL_MAX_IMAGES, FILL_MAX_TEXT, fillSummary, takesImages } from './fill';

  // The Describe it box (WM-FILL-6): a sentence and/or images go to the
  // engine, which answers with checked values; the form merges them and the
  // user saves as usual. Nothing here writes.

  interface Props {
    form: FillFormKey;
    /** What the form holds now, sent for edits (WM-FILL-4). */
    current?: () => FillValues;
    onfill: (result: FillResult) => void;
    /** Names a row in the summary, e.g. "limit". */
    rowsLabel?: string;
    /** Starts with this text, e.g. from the chat. */
    initialText?: string;
    class?: string;
  }

  let { form, current, onfill, rowsLabel = 'row', initialText = '', class: className = '' }: Props = $props();

  /** The engine's default per-image limit (WM-CAP-1); larger images are downscaled. */
  const MAX_IMAGE_BYTES = 5 * 1024 * 1024;

  interface Picked {
    id: number;
    name: string;
    blob: Blob;
    mediaType: string;
    url: string;
  }

  // svelte-ignore state_referenced_locally
  let text = $state(initialText);
  let images = $state<Picked[]>([]);
  let busy = $state(false);
  let summary = $state('');
  let notes = $state<FillNote[]>([]);
  let problem = $state('');
  let dragging = $state(false);
  let nextId = 0;
  let controller: AbortController | null = null;
  let picker: HTMLInputElement | undefined = $state();

  const allowImages = $derived(takesImages(form));
  const id = $derived(`fill-${form}`);
  const canFill = $derived(!busy && (text.trim() !== '' || images.length > 0) && text.length <= FILL_MAX_TEXT);

  async function addFiles(files: Iterable<File>) {
    if (!allowImages) return;
    problem = '';
    for (const file of files) {
      if (!isImageFile(file)) continue;
      if (images.length >= FILL_MAX_IMAGES) {
        problem = `At most ${FILL_MAX_IMAGES} images.`;
        break;
      }
      try {
        const { blob, mediaType } = await prepareImage(file, MAX_IMAGE_BYTES);
        images = [
          ...images,
          { id: nextId++, name: file.name || 'image', blob, mediaType, url: URL.createObjectURL(blob) },
        ];
      } catch (cause) {
        problem = cause instanceof Error ? cause.message : 'The image could not be read.';
      }
    }
  }

  function remove(picked: Picked) {
    URL.revokeObjectURL(picked.url);
    images = images.filter((image) => image.id !== picked.id);
  }

  async function fill() {
    if (!canFill) return;
    controller?.abort();
    controller = new AbortController();
    busy = true;
    problem = '';
    try {
      const payload = await Promise.all(
        images.map(async (image) => ({
          mediaType: image.mediaType,
          data: toBase64(new Uint8Array(await image.blob.arrayBuffer())),
        })),
      );
      const result = await api.fill(
        {
          form,
          text: text.trim() || undefined,
          images: payload.length ? payload : undefined,
          current: current?.(),
        },
        controller.signal,
      );
      notes = result.notes;
      summary = fillSummary(result, rowsLabel);
      onfill(result);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === 'AbortError') return;
      summary = '';
      notes = [];
      problem = cause instanceof Error ? cause.message : 'Could not fill the form.';
    } finally {
      busy = false;
    }
  }

  function onkeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void fill();
    }
  }

  function onpaste(event: ClipboardEvent) {
    const files = [...(event.clipboardData?.files ?? [])].filter(isImageFile);
    if (!allowImages || !files.length) return;
    event.preventDefault();
    void addFiles(files);
  }

  function ondrop(event: DragEvent) {
    dragging = false;
    if (!allowImages || !event.dataTransfer?.files.length) return;
    event.preventDefault();
    void addFiles(event.dataTransfer.files);
  }

  onDestroy(() => {
    controller?.abort();
    for (const image of images) URL.revokeObjectURL(image.url);
  });
</script>

<section
  class="rounded-sm border border-dashed border-accent-400/70 bg-accent-50/40 p-3 dark:border-accent-500/50 dark:bg-accent-950/20 {dragging
    ? 'ring-2 ring-accent-400'
    : ''} {className}"
  aria-labelledby="{id}-label"
  ondragover={(event) => {
    if (!allowImages) return;
    event.preventDefault();
    dragging = true;
  }}
  ondragleave={() => (dragging = false)}
  {ondrop}
>
  <label id="{id}-label" for="{id}-text" class="label flex items-center gap-1">
    <Icon name="sparkles" size={13} />
    Describe it
  </label>
  <textarea
    id="{id}-text"
    class="field mt-1 min-h-[3.25rem] resize-y"
    rows="2"
    maxlength={FILL_MAX_TEXT}
    placeholder={FILL_EXAMPLES[form]}
    bind:value={text}
    {onkeydown}
    {onpaste}
    aria-describedby="{id}-hint"
  ></textarea>
  {#if images.length}
    <ul class="mt-2 flex flex-wrap gap-2" aria-label="Images to read">
      {#each images as image (image.id)}
        <li class="relative">
          <img src={image.url} alt={image.name} class="h-14 w-14 rounded-sm border border-edge object-cover" />
          <button
            type="button"
            class="icon-btn absolute -top-2 -right-2 rounded-full bg-surface p-0.5 shadow-hard"
            aria-label="Remove {image.name}"
            onclick={() => remove(image)}><Icon name="x" size={12} /></button
          >
        </li>
      {/each}
    </ul>
  {/if}
  <div class="mt-2 flex flex-wrap items-center gap-2">
    <Button size="sm" variant="primary" icon="sparkles" {busy} busyText="Filling…" disabled={!canFill} onclick={fill}
      >Fill</Button
    >
    {#if allowImages}
      <Button size="sm" icon="image" disabled={busy || images.length >= FILL_MAX_IMAGES} onclick={() => picker?.click()}
        >Add image</Button
      >
      <input
        bind:this={picker}
        type="file"
        accept="image/*,.heic,.heif"
        multiple
        class="hidden"
        aria-label="Pick images"
        onchange={(event) => {
          const input = event.currentTarget;
          void addFiles([...(input.files ?? [])]);
          input.value = '';
        }}
      />
    {/if}
    <p id="{id}-hint" class="muted min-w-0 flex-1 text-xs">
      {allowImages ? 'Type, paste or drop an image. ' : ''}AI fills the form; nothing is saved until you save.
    </p>
  </div>
  <div aria-live="polite">
    {#if problem}
      <p class="mt-2 text-xs text-rose-600 dark:text-rose-400">{problem}</p>
    {:else if summary}
      <p class="mt-2 text-xs font-medium">{summary}</p>
    {/if}
    {#if notes.length}
      <ul class="mt-1 list-disc space-y-0.5 pl-4 text-xs text-amber-700 dark:text-amber-300">
        {#each notes as note, index (index)}
          <li>
            {#if note.label && !note.message.startsWith(note.label)}<span class="font-medium"
                >{note.label}{note.row !== undefined ? ` (${rowsLabel} ${note.row + 1})` : ''}:</span
              >{' '}{/if}{note.message}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>
