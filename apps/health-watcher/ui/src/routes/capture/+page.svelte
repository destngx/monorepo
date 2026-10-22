<script lang="ts">
  import { onMount } from 'svelte';
  import { CaptureError, captureTarget, sendFromPhone, sentMessage, takeCaptureToken } from '$lib/capture';
  import { draftStore, idbBackend } from '$lib/drafts';
  import Icon from '$lib/components/Icon.svelte';
  import { formatSize } from '$lib/format';
  import { ACCEPT, pickFiles } from '$lib/upload';

  // The phone end of a capture link (HW-IMP-9): no sign-in, only a camera
  // and a short-lived token that adds to one inbox, or to one item when the
  // link was made on an item's page (HW-IMP-10). Photos wait in a draft, so
  // a reload while the camera is open loses nothing.
  interface Staged {
    file: File;
    url: string;
  }

  const draftKey = () => (toItem ? 'capture-item' : 'capture');
  const drafts = draftStore(idbBackend());

  let token = $state<string | null>(null);
  let staged = $state<Staged[]>([]);
  let oneItem = $state(true);
  let toItem = $state(false);
  let ready = $state(false);
  let sending = $state(false);
  let message = $state<{ tone: 'ok' | 'error'; text: string } | null>(null);
  let expired = $state(false);
  let cameraInput: HTMLInputElement | undefined = $state();
  let filesInput: HTMLInputElement | undefined = $state();

  onMount(() => {
    token = takeCaptureToken(location.hash, location.pathname, sessionStorage, (url) =>
      history.replaceState(history.state, '', url),
    );
    const target = token
      ? captureTarget(token).catch((error: unknown) => {
          message = { tone: 'error', text: error instanceof Error ? error.message : String(error) };
          return { item: false };
        })
      : Promise.resolve({ item: false });
    target
      .then((t) => {
        if (t === null) {
          expired = true;
          message = { tone: 'error', text: 'This link has expired. Make a new QR code on the Mac.' };
        } else {
          toItem = t.item;
        }
        return drafts.load(draftKey());
      })
      .then((d) => {
        if (d) {
          staged = d.files.map((file) => ({ file, url: URL.createObjectURL(file) }));
          oneItem = d.oneItem;
        }
      })
      .catch((error: unknown) => {
        message = { tone: 'error', text: `Could not bring back earlier photos: ${String(error)}` };
      })
      .finally(() => (ready = true));
    return () => {
      for (const s of staged) URL.revokeObjectURL(s.url);
    };
  });

  $effect(() => {
    if (!ready) return;
    const draft = { files: staged.map((s) => s.file), oneItem, note: '', personId: '' };
    drafts.save(draftKey(), draft).catch((error: unknown) => {
      message = { tone: 'error', text: `Could not keep the photos for later: ${String(error)}` };
    });
  });

  function add(files: FileList | null) {
    if (!files) return;
    const { accepted, skipped } = pickFiles(
      files,
      staged.map((s) => s.file),
    );
    staged = [...staged, ...accepted.map((file) => ({ file, url: URL.createObjectURL(file) }))];
    message = skipped.length ? { tone: 'error', text: `Skipped: ${skipped.join(', ')}` } : null;
  }

  function remove(i: number) {
    URL.revokeObjectURL(staged[i].url);
    staged = staged.filter((_, j) => j !== i);
  }

  async function send() {
    if (!token || !staged.length || sending) return;
    sending = true;
    message = null;
    try {
      const res = await sendFromPhone(
        token,
        staged.map((s) => s.file),
        oneItem && !toItem,
      );
      for (const s of staged) URL.revokeObjectURL(s.url);
      staged = [];
      message = { tone: 'ok', text: sentMessage(res, toItem) };
    } catch (error) {
      expired = error instanceof CaptureError && error.expired;
      message = { tone: 'error', text: error instanceof Error ? error.message : String(error) };
    } finally {
      sending = false;
    }
  }

  const total = $derived(staged.reduce((sum, s) => sum + s.file.size, 0));
</script>

<svelte:head><title>Phone camera · Health Watcher</title></svelte:head>

<main class="mx-auto flex min-h-screen max-w-md flex-col gap-4 px-4 py-6">
  <header class="flex items-center gap-2">
    <span class="grid size-9 place-items-center rounded-xl bg-accent-600 text-ink-50"
      ><Icon name="health" size={18} /></span
    >
    <div>
      <h1 class="font-semibold">Phone camera</h1>
      <p class="muted text-xs">
        {toItem
          ? 'Photos are added to one item in the Health Watcher inbox on the Mac.'
          : 'Photos go to the Health Watcher inbox on the Mac.'}
      </p>
    </div>
  </header>

  {#if !token}
    <section class="panel p-5 text-sm">
      This page opens from the QR code on the Mac: <em>Take photos with iPhone</em> in the Inbox or on an item.
    </section>
  {:else}
    <section class="panel flex flex-col gap-3 p-5">
      <button class="btn-primary py-4 text-base" type="button" onclick={() => cameraInput?.click()} disabled={expired}
        ><Icon name="camera" size={20} />Take a photo</button
      >
      <button class="btn-ghost" type="button" onclick={() => filesInput?.click()} disabled={expired}
        ><Icon name="image" size={16} />Choose from Photos or Files</button
      >
      <p class="muted text-xs">One photo per page. Lay the paper flat in good light.</p>
    </section>

    {#if staged.length}
      <section class="panel p-4">
        <div class="grid grid-cols-3 gap-2">
          {#each staged as s, i (s.url)}
            <div
              class="relative aspect-[3/4] overflow-hidden rounded-xl border border-ink-900/10 dark:border-ink-50/10"
            >
              {#if s.file.type.startsWith('image/') && s.file.type !== 'image/heic'}
                <img src={s.url} alt="Page {i + 1}" class="size-full object-cover" />
              {:else}
                <div class="grid size-full place-items-center p-2 text-center text-xs break-all">{s.file.name}</div>
              {/if}
              <button
                type="button"
                class="icon-btn absolute top-1 right-1 size-7 bg-ink-50/90 dark:bg-ink-900/90"
                aria-label="Remove page {i + 1}"
                onclick={() => remove(i)}><Icon name="x" size={14} /></button
              >
            </div>
          {/each}
        </div>
        {#if staged.length > 1 && !toItem}
          <label class="mt-3 flex items-center gap-2 text-sm">
            <input type="checkbox" bind:checked={oneItem} /> These are pages of one paper
          </label>
        {/if}
        <button class="btn-primary mt-4 w-full" type="button" onclick={() => void send()} disabled={sending || expired}>
          {sending
            ? 'Sending…'
            : `Send ${staged.length} ${staged.length === 1 ? 'page' : 'pages'} (${formatSize(total)})`}
        </button>
      </section>
    {/if}
  {/if}

  {#if message}
    <p
      class="panel p-4 text-sm {message.tone === 'ok'
        ? 'text-accent-700 dark:text-accent-300'
        : 'text-danger-600 dark:text-danger-400'}"
      role={message.tone === 'ok' ? 'status' : 'alert'}
    >
      {message.text}
    </p>
  {/if}

  <input
    bind:this={cameraInput}
    class="hidden"
    type="file"
    accept="image/*"
    capture="environment"
    onchange={(e) => {
      add(e.currentTarget.files);
      e.currentTarget.value = '';
    }}
  />
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
</main>
