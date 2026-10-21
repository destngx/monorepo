<script lang="ts">
  import type { Person, UploadResponse } from '$lib/api/types';
  import { formatSize } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { ACCEPT, pickFiles } from '$lib/upload';
  import Icon from './Icon.svelte';

  // Staging area for uploads (HW-IMP-3, HW-IMP-5): files from the picker, a
  // folder, drag and drop or the phone camera wait here until sent. In
  // backlog mode every file belongs to one chosen person (HW-IMP-6).
  let { people = [], onuploaded }: { people?: Person[]; onuploaded: (resp: UploadResponse) => void } = $props();

  interface Staged {
    file: File;
    url: string;
  }

  let staged = $state<Staged[]>([]);
  let oneItem = $state(false);
  let note = $state('');
  let personId = $state('');
  let dragging = $state(false);
  let filesInput: HTMLInputElement | undefined = $state();
  let folderInput: HTMLInputElement | undefined = $state();
  let cameraInput: HTMLInputElement | undefined = $state();

  const totalSize = $derived(staged.reduce((sum, s) => sum + s.file.size, 0));

  function add(files: FileList | File[] | null, fromCamera = false) {
    if (!files) return;
    const { accepted, skipped } = pickFiles(
      files,
      staged.map((s) => s.file),
    );
    staged = [...staged, ...accepted.map((file) => ({ file, url: URL.createObjectURL(file) }))];
    // Photos taken one after another are usually the pages of one paper.
    if (fromCamera && staged.length > 1) oneItem = true;
    if (skipped.length) toasts.push('warning', `Skipped: ${skipped.join(', ')}`);
  }

  function remove(i: number) {
    URL.revokeObjectURL(staged[i].url);
    staged = staged.filter((_, j) => j !== i);
  }

  function clear() {
    for (const s of staged) URL.revokeObjectURL(s.url);
    staged = [];
    oneItem = false;
    note = '';
  }

  $effect(() => () => {
    for (const s of staged) URL.revokeObjectURL(s.url);
  });

  // webkitdirectory isn't in Svelte's attribute types; set it on the element.
  $effect(() => {
    folderInput?.setAttribute('webkitdirectory', '');
  });

  const send = action(
    async () => {
      const resp = await session.api.upload(
        staged.map((s) => s.file),
        { oneItem: oneItem && staged.length > 1, note: note.trim(), personId },
      );
      const dups = resp.files.filter((f) => f.duplicate).length;
      const added = resp.items.length;
      toasts.success(
        `${added} ${added === 1 ? 'item' : 'items'} added to the inbox` +
          (dups ? `; ${dups} already stored, skipped` : ''),
      );
      clear();
      onuploaded(resp);
    },
    { errorContext: 'Upload failed' },
  );
</script>

<section
  class="glass border-2 border-dashed p-5 transition-colors {dragging
    ? 'border-teal-500 bg-teal-500/5'
    : 'border-transparent'}"
  aria-label="Upload"
  ondragover={(e) => {
    e.preventDefault();
    dragging = true;
  }}
  ondragleave={() => (dragging = false)}
  ondrop={(e) => {
    e.preventDefault();
    dragging = false;
    add(e.dataTransfer?.files ?? null);
  }}
>
  <div class="flex flex-col items-center gap-3 text-center">
    <span class="grid size-12 place-items-center rounded-2xl bg-teal-600/10 text-teal-700 dark:text-teal-300"
      ><Icon name="upload" size={24} /></span
    >
    <div>
      <p class="font-medium">Drop PDFs or photos here</p>
      <p class="muted text-sm">Up to 50 MB a file. Duplicates are recognised and skipped.</p>
    </div>
    <div class="flex flex-wrap justify-center gap-2">
      <button class="btn-primary" type="button" onclick={() => cameraInput?.click()}
        ><Icon name="camera" size={16} />Take photos</button
      >
      <button class="btn-ghost" type="button" onclick={() => filesInput?.click()}
        ><Icon name="file" size={16} />Choose files</button
      >
      <button class="btn-ghost hidden sm:inline-flex" type="button" onclick={() => folderInput?.click()}>
        <Icon name="folder" size={16} />Choose a folder
      </button>
    </div>
  </div>
  <input
    bind:this={filesInput}
    class="hidden"
    type="file"
    multiple
    accept={ACCEPT}
    onchange={(e) => {
      add(e.currentTarget.files);
      e.currentTarget.value = '';
    }}
  />
  <input
    bind:this={folderInput}
    class="hidden"
    type="file"
    multiple
    onchange={(e) => {
      add(e.currentTarget.files);
      e.currentTarget.value = '';
    }}
  />
  <input
    bind:this={cameraInput}
    class="hidden"
    type="file"
    accept="image/*"
    capture="environment"
    onchange={(e) => {
      add(e.currentTarget.files, true);
      e.currentTarget.value = '';
    }}
  />

  {#if staged.length}
    <div class="mt-5 border-t border-slate-900/10 pt-4 dark:border-white/10">
      <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
        {#each staged as s, i (s.url)}
          <div
            class="relative aspect-[3/4] overflow-hidden rounded-xl border border-slate-900/10 bg-white/70 dark:border-white/10 dark:bg-slate-900/50"
          >
            {#if s.file.type.startsWith('image/') && s.file.type !== 'image/heic'}
              <img src={s.url} alt={s.file.name} class="size-full object-cover" />
            {:else}
              <span class="grid size-full place-items-center p-2 text-center">
                <span class="muted line-clamp-3 text-[11px] break-all">{s.file.name}</span>
              </span>
            {/if}
            {#if oneItem && staged.length > 1}
              <span class="absolute top-1 left-1 rounded-md bg-slate-900/70 px-1.5 text-[11px] text-white">{i + 1}</span
              >
            {/if}
            <button
              type="button"
              class="absolute top-1 right-1 grid size-6 place-items-center rounded-full bg-slate-900/70 text-white"
              aria-label="Remove {s.file.name}"
              onclick={() => remove(i)}><Icon name="x" size={12} /></button
            >
          </div>
        {/each}
        <button
          type="button"
          class="muted grid aspect-[3/4] place-items-center rounded-xl border border-dashed border-slate-400/60 text-xs"
          onclick={() => cameraInput?.click()}
        >
          <span class="flex flex-col items-center gap-1"><Icon name="plus" size={18} />Add page</span>
        </button>
      </div>
      {#if staged.length > 1}
        <label class="mt-4 flex items-center gap-2 text-sm">
          <input type="checkbox" bind:checked={oneItem} />
          These {staged.length} files are pages of one document
        </label>
      {/if}
      <label class="label mt-4" for="upload-note">Note (optional)</label>
      <input
        id="upload-note"
        class="field"
        bind:value={note}
        maxlength="2000"
        placeholder="e.g. Dentist, March check-up"
      />
      {#if people.length > 1}
        <label class="label mt-4" for="upload-person">Whose papers (optional)</label>
        <select id="upload-person" class="field" bind:value={personId}>
          <option value="">Find out from the papers</option>
          {#each people as p (p.id)}<option value={p.id}>{p.displayName}</option>{/each}
        </select>
        <p class="muted mt-1 text-xs">
          For a pile of old papers of one person. They are grouped into visits by date and place once read.
        </p>
      {/if}
      <div class="mt-4 flex items-center justify-between gap-2">
        <span class="muted text-sm"
          >{staged.length} {staged.length === 1 ? 'file' : 'files'} · {formatSize(totalSize)}</span
        >
        <span class="flex gap-2">
          <button class="btn-ghost" type="button" onclick={clear} disabled={send.busy}>Clear</button>
          <button
            class="btn-primary"
            type="button"
            onclick={() => void send.run()}
            disabled={send.busy}
            aria-busy={send.busy || undefined}
          >
            {send.busy ? 'Uploading…' : 'Upload'}
          </button>
        </span>
      </div>
    </div>
  {/if}
</section>
