<script lang="ts">
  import { draftStore, idbBackend } from '$lib/drafts';
  import { formatSize } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { pickFiles } from '$lib/upload';
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  // The phone camera on an unfiled item (HW-IMP-10). The camera gives one
  // photo a shot, so the photos wait here and go to the item together: one
  // new reading, not one per photo. They are kept in IndexedDB, because iOS
  // can unload the page while the camera is open.
  interface Props {
    itemId: string;
    busy: boolean;
    onsend: (files: File[]) => Promise<boolean>;
  }

  let { itemId, busy, onsend }: Props = $props();

  interface Staged {
    file: File;
    url: string;
  }

  let staged = $state<Staged[]>([]);
  let input: HTMLInputElement | undefined = $state();
  let restored = $state<string | null>(null);
  const drafts = draftStore(idbBackend());
  const key = $derived(`item-photos:${itemId}`);

  /** Opens the camera for one more photo. */
  export function take() {
    input?.click();
  }

  $effect(() => {
    const k = key;
    if (restored === k) return;
    drafts
      .load(k)
      .then((d) => {
        if (d && !staged.length) {
          staged = d.files.map((file) => ({ file, url: URL.createObjectURL(file) }));
          toasts.info(`${d.files.length} ${d.files.length === 1 ? 'photo' : 'photos'} not sent yet are back`);
        }
      })
      .catch((error: unknown) => toasts.error(error, 'Could not bring back the photos not sent yet'))
      .finally(() => (restored = k));
  });
  $effect(() => {
    const k = key;
    if (restored !== k) return;
    const draft = { files: staged.map((s) => s.file), oneItem: true, note: '', personId: '' };
    drafts.save(k, draft).catch((error: unknown) => toasts.error(error, 'Could not keep the photos for later'));
  });
  $effect(() => () => {
    for (const s of staged) URL.revokeObjectURL(s.url);
  });

  function add(files: FileList | null) {
    if (!files) return;
    const { accepted, skipped } = pickFiles(
      files,
      staged.map((s) => s.file),
    );
    staged = [...staged, ...accepted.map((file) => ({ file, url: URL.createObjectURL(file) }))];
    if (skipped.length) toasts.push('warning', `Skipped: ${skipped.join(', ')}`);
  }

  function remove(i: number) {
    URL.revokeObjectURL(staged[i].url);
    staged = staged.filter((_, j) => j !== i);
  }

  async function send() {
    if (!(await onsend(staged.map((s) => s.file)))) return;
    for (const s of staged) URL.revokeObjectURL(s.url);
    staged = [];
  }

  const total = $derived(staged.reduce((sum, s) => sum + s.file.size, 0));
</script>

{#if staged.length}
  <section class="panel mb-5 p-4" aria-label="Photos to add">
    <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
      {#each staged as s, i (s.url)}
        <div class="relative aspect-[3/4] overflow-hidden rounded-xl border border-ink-900/10 dark:border-ink-50/10">
          {#if s.file.type.startsWith('image/') && s.file.type !== 'image/heic'}
            <img src={s.url} alt="Photo {i + 1}" class="size-full object-cover" />
          {:else}
            <div class="grid size-full place-items-center p-2 text-center text-xs break-all">{s.file.name}</div>
          {/if}
          <button
            type="button"
            class="icon-btn absolute top-1 right-1 size-7 bg-ink-50/90 dark:bg-ink-900/90"
            aria-label="Remove photo {i + 1}"
            onclick={() => remove(i)}><Icon name="x" size={14} /></button
          >
        </div>
      {/each}
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      <Button variant="primary" icon="plus" {busy} busyText="Adding…" onclick={send}
        >Add {staged.length} {staged.length === 1 ? 'photo' : 'photos'} ({formatSize(total)})</Button
      >
      <Button icon="camera" onclick={take}>Take another</Button>
    </div>
  </section>
{/if}

<input
  bind:this={input}
  class="hidden"
  type="file"
  accept="image/*"
  capture="environment"
  onchange={(e) => {
    add(e.currentTarget.files);
    e.currentTarget.value = '';
  }}
/>
