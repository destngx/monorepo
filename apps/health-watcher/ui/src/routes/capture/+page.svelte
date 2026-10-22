<script lang="ts">
  import { onMount } from 'svelte';
  import {
    CaptureError,
    captureTarget,
    sendFromPhone,
    sentMessage,
    stageFromPhone,
    takeCaptureToken,
  } from '$lib/capture';
  import { holdDevReload } from '$lib/devReload';
  import { draftStore, idbBackend } from '$lib/drafts';
  import Icon from '$lib/components/Icon.svelte';
  import QueueThumbs from '$lib/components/QueueThumbs.svelte';
  import { formatSize } from '$lib/format';
  import { UploadQueue } from '$lib/uploadQueue.svelte';
  import { ACCEPT, pickFiles } from '$lib/upload';

  // The phone end of a capture link (HW-IMP-9): no sign-in, only a camera
  // and a short-lived token that adds to one inbox, or to one item when the
  // link was made on an item's page (HW-IMP-10). Photos stay on the phone
  // until Send, which uploads them with a bar each (HW-IMP-11). They also
  // wait in a draft, so a reload while the camera is open loses nothing.
  const draftKey = () => (toItem ? 'capture-item' : 'capture');
  const drafts = draftStore(idbBackend());

  let token = $state<string | null>(null);
  let oneItem = $state(true);
  let toItem = $state(false);
  let ready = $state(false);
  let sending = $state(false);
  let message = $state<{ tone: 'ok' | 'error'; text: string } | null>(null);
  let expired = $state(false);
  let cameraInput: HTMLInputElement | undefined = $state();
  let filesInput: HTMLInputElement | undefined = $state();

  const queue = new UploadQueue(async (file, progress) => {
    try {
      return await stageFromPhone(token ?? '', file, progress);
    } catch (error) {
      if (error instanceof CaptureError && error.expired) expired = true;
      throw error;
    }
  });

  onMount(() => {
    const release = holdDevReload();
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
          oneItem = d.oneItem;
          if (!expired) queue.add(d.files);
        }
      })
      .catch((error: unknown) => {
        message = { tone: 'error', text: `Could not bring back earlier photos: ${String(error)}` };
      })
      .finally(() => (ready = true));
    return () => {
      release();
      queue.clear();
    };
  });

  $effect(() => {
    if (!ready) return;
    const draft = { files: queue.entries.map((e) => e.file), oneItem, note: '', personId: '' };
    drafts.save(draftKey(), draft).catch((error: unknown) => {
      message = { tone: 'error', text: `Could not keep the photos for later: ${String(error)}` };
    });
  });

  function add(files: FileList | null) {
    if (!files) return;
    const { accepted, skipped } = pickFiles(
      files,
      queue.entries.map((e) => e.file),
    );
    queue.add(accepted);
    message = skipped.length ? { tone: 'error', text: `Skipped: ${skipped.join(', ')}` } : null;
  }

  async function send() {
    if (!token || !queue.entries.length || sending) return;
    sending = true;
    message = null;
    try {
      await queue.upload();
      if (queue.failed) {
        const n = queue.failed;
        message = {
          tone: 'error',
          text: `${n} ${n === 1 ? 'photo' : 'photos'} did not reach the Mac. Press Send to try again.`,
        };
        return;
      }
      const ids = queue.fileIds;
      if (!ids.length) {
        message = { tone: 'ok', text: 'These photos are already stored on the Mac.' };
        queue.clear();
        return;
      }
      const res = await sendFromPhone(token, ids, oneItem && !toItem);
      message = {
        tone: 'ok',
        text: sentMessage({ added: res.added, files: ids.length, duplicates: queue.duplicates }, toItem),
      };
      queue.clear();
    } catch (error) {
      expired = error instanceof CaptureError && error.expired;
      message = { tone: 'error', text: error instanceof Error ? error.message : String(error) };
    } finally {
      sending = false;
    }
  }

  const total = $derived(queue.entries.reduce((sum, e) => sum + e.file.size, 0));
  const uploaded = $derived(queue.entries.reduce((sum, e) => sum + e.file.size * e.progress, 0));
  const sendLabel = $derived.by(() => {
    const n = queue.entries.length;
    if (queue.pending) return `Sending… ${formatSize(uploaded)} of ${formatSize(total)}`;
    if (sending) return 'Sending…';
    if (queue.failed) return `Send again (${queue.failed} failed)`;
    return `Send ${n} ${n === 1 ? 'page' : 'pages'} (${formatSize(total)})`;
  });
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

    {#if queue.entries.length}
      <section class="panel p-4">
        <QueueThumbs {queue} disabled={sending} />
        {#if queue.entries.length > 1 && !toItem}
          <label class="mt-3 flex items-center gap-2 text-sm">
            <input type="checkbox" bind:checked={oneItem} /> These are pages of one paper
          </label>
        {/if}
        <button
          class="btn-primary mt-4 w-full"
          type="button"
          onclick={() => void send()}
          disabled={sending || expired}
          aria-busy={sending}>{sendLabel}</button
        >
        <p class="muted mt-2 text-xs">The photos stay on this phone until you press Send.</p>
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
