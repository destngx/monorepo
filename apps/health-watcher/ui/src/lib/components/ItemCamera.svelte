<script lang="ts">
  import { holdDevReload } from '$lib/devReload';
  import { draftStore, idbBackend } from '$lib/drafts';
  import { formatSize } from '$lib/format';
  import { session } from '$lib/state/session.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { UploadQueue } from '$lib/uploadQueue.svelte';
  import { pickFiles } from '$lib/upload';
  import Button from './Button.svelte';
  import QueueThumbs from './QueueThumbs.svelte';

  // The phone camera on an unfiled item (HW-IMP-10). Photos stay on the
  // phone until Add, which uploads them with a bar each (HW-IMP-11) and puts
  // them all in the item at once, so it is read once. They are also kept in
  // IndexedDB, because iOS can unload the page while the camera is open.
  interface Props {
    itemId: string;
    busy: boolean;
    onsend: (fileIds: string[]) => Promise<boolean>;
  }

  let { itemId, busy, onsend }: Props = $props();

  let input: HTMLInputElement | undefined = $state();
  let restored = $state<string | null>(null);
  let uploading = $state(false);
  const drafts = draftStore(idbBackend());
  const key = $derived(`item-photos:${itemId}`);
  const queue = new UploadQueue((file, progress) => session.api.stage(file, progress));

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
        if (d && !queue.entries.length) {
          queue.add(d.files);
          toasts.info(`${d.files.length} ${d.files.length === 1 ? 'photo' : 'photos'} not added yet are back`);
        }
      })
      .catch((error: unknown) => toasts.error(error, 'Could not bring back the photos not added yet'))
      .finally(() => (restored = k));
  });
  $effect(() => {
    const k = key;
    if (restored !== k) return;
    const draft = { files: queue.entries.map((e) => e.file), oneItem: true, note: '', personId: '' };
    drafts.save(k, draft).catch((error: unknown) => toasts.error(error, 'Could not keep the photos for later'));
  });
  $effect(() => () => queue.clear());
  $effect(() => holdDevReload());

  function add(files: FileList | null) {
    if (!files) return;
    const { accepted, skipped } = pickFiles(
      files,
      queue.entries.map((e) => e.file),
    );
    queue.add(accepted);
    if (skipped.length) toasts.push('warning', `Skipped: ${skipped.join(', ')}`);
  }

  async function send() {
    if (!queue.entries.length || uploading) return;
    uploading = true;
    try {
      await queue.upload();
    } finally {
      uploading = false;
    }
    if (queue.failed) {
      toasts.push(
        'warning',
        `${queue.failed} ${queue.failed === 1 ? 'photo' : 'photos'} did not reach the Mac: press Add to try again`,
      );
      return;
    }
    if (!queue.fileIds.length) {
      toasts.info('These photos are already stored');
      queue.clear();
      return;
    }
    if (await onsend(queue.fileIds)) queue.clear();
  }

  const total = $derived(queue.entries.reduce((sum, e) => sum + e.file.size, 0));
  const uploaded = $derived(queue.entries.reduce((sum, e) => sum + e.file.size * e.progress, 0));
  const label = $derived.by(() => {
    const n = queue.entries.length;
    if (uploading) return `Uploading… ${formatSize(uploaded)} of ${formatSize(total)}`;
    if (busy) return 'Adding…';
    if (queue.failed) return `Add again (${queue.failed} failed)`;
    return `Add ${n} ${n === 1 ? 'photo' : 'photos'} to this item`;
  });
</script>

{#if queue.entries.length}
  <section class="panel mb-5 p-4" aria-label="Photos to add">
    <QueueThumbs {queue} disabled={busy || uploading} />
    <div class="mt-4 flex flex-wrap gap-2">
      <Button variant="primary" icon="plus" busy={busy || uploading} onclick={send}>{label}</Button>
      <Button icon="camera" disabled={busy || uploading} onclick={take}>Take another</Button>
    </div>
    <p class="muted mt-2 text-xs">The photos stay on this phone until you press Add.</p>
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
