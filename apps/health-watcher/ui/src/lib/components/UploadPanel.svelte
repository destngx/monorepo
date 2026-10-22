<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import type { Person, UploadResponse } from '$lib/api/types';
  import { formatSize } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { draftStore, idbBackend } from '$lib/drafts';
  import { ACCEPT, pickFiles } from '$lib/upload';
  import FilmGuide from './FilmGuide.svelte';
  import Icon from './Icon.svelte';
  import PhoneCaptureModal from './PhoneCaptureModal.svelte';

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
  let phoneOpen = $state(false);

  // Staged files survive a reload: iOS can unload the page while the camera
  // is open. Restore once per tenant, then keep the draft in step.
  const drafts = draftStore(idbBackend());
  const draftKey = $derived(session.tenantId ? `upload:${session.tenantId}` : null);
  let restoredKey = $state<string | null>(null);
  $effect(() => {
    const key = draftKey;
    if (!key || restoredKey === key) return;
    drafts
      .load(key)
      .then((d) => {
        if (d && !staged.length) {
          staged = d.files.map((file) => ({ file, url: URL.createObjectURL(file) }));
          oneItem = d.oneItem;
          note = d.note;
          personId = d.personId;
          toasts.info(`${d.files.length} ${d.files.length === 1 ? 'file' : 'files'} not sent yet are back`);
        }
      })
      .catch((error: unknown) => toasts.error(error, 'Could not bring back the files not sent yet'))
      .finally(() => (restoredKey = key));
  });
  $effect(() => {
    const key = draftKey;
    if (!key || restoredKey !== key) return;
    const draft = { files: staged.map((s) => s.file), oneItem, note, personId };
    drafts.save(key, draft).catch((error: unknown) => toasts.error(error, 'Could not keep the files for later'));
  });

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
  class="panel border-2 border-dashed p-5 transition-colors {dragging ? 'border-accent-600 bg-accent-500/5' : ''}"
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
    <span class="grid size-12 place-items-center rounded-2xl bg-accent-600/10 text-accent-700 dark:text-accent-300"
      ><Icon name="upload" size={24} /></span
    >
    <div>
      <p class="font-medium">Drop PDFs or photos here</p>
      <p class="muted text-sm">Up to 50 MB a file. Duplicates are recognised and skipped.</p>
    </div>
    <div class="flex flex-wrap justify-center gap-2">
      <!-- A phone opens its own camera; a computer shows a QR code for the phone. -->
      <button class="btn-primary hidden pointer-coarse:inline-flex" type="button" onclick={() => cameraInput?.click()}
        ><Icon name="camera" size={16} />Take photos</button
      >
      <button class="btn-primary pointer-coarse:hidden" type="button" onclick={() => (phoneOpen = true)}
        ><Icon name="camera" size={16} />Take photos with iPhone</button
      >
      <button class="btn-ghost" type="button" onclick={() => filesInput?.click()}
        ><Icon name="file" size={16} />Choose files</button
      >
      <button class="btn-ghost hidden sm:inline-flex" type="button" onclick={() => folderInput?.click()}>
        <Icon name="folder" size={16} />Choose a folder
      </button>
    </div>
    <div class="w-full max-w-lg"><FilmGuide /></div>
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
    <div class="mt-5 border-t border-ink-900/10 pt-4 dark:border-ink-50/10">
      <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
        {#each staged as s, i (s.url)}
          <div
            class="relative aspect-[3/4] overflow-hidden rounded-xl border border-ink-900/10 bg-ink-50/70 dark:border-ink-50/10 dark:bg-ink-900/50"
          >
            {#if s.file.type.startsWith('image/') && s.file.type !== 'image/heic'}
              <img src={s.url} alt={s.file.name} class="size-full object-cover" />
            {:else}
              <span class="grid size-full place-items-center p-2 text-center">
                <span class="muted line-clamp-3 text-[11px] break-all">{s.file.name}</span>
              </span>
            {/if}
            {#if oneItem && staged.length > 1}
              <span class="absolute top-1 left-1 rounded-md bg-ink-900/70 px-1.5 text-[11px] text-ink-50">{i + 1}</span>
            {/if}
            <button
              type="button"
              class="absolute top-1 right-1 grid size-6 place-items-center rounded-full bg-ink-900/70 text-ink-50"
              aria-label="Remove {s.file.name}"
              onclick={() => remove(i)}><Icon name="x" size={12} /></button
            >
          </div>
        {/each}
        <button
          type="button"
          class="muted grid aspect-[3/4] place-items-center rounded-xl border border-dashed border-ink-400/60 text-xs"
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

<PhoneCaptureModal
  open={phoneOpen}
  onclose={() => {
    phoneOpen = false;
    void invalidateAll();
  }}
/>
