<script lang="ts">
  import { untrack } from 'svelte';
  import { invalidate } from '$app/navigation';
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import type { QuickCapture, QuickCaptureOverview } from '$lib/api/types';
  import CaptureReview from '$lib/capture/CaptureReview.svelte';
  import { captureStatus, draftItems, draftResult, mergeDraftRows, toDraftRows } from '$lib/capture/drafts';
  import { isImageFile, prepareImage } from '$lib/capture/materials';
  import type { ReviewRow } from '$lib/capture/review';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatDateTime } from '$lib/format';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  // Quick capture drafts (WM-CAP-19): screenshots sent from the phone are read
  // in the background; nothing reaches the sheet until a draft is approved.
  let { data } = $props();

  const POLL_MS = 5000;
  const DEFAULT_ITEM_BYTES = 5 * 1024 * 1024;

  const result = $derived(draftResult(data.drafts));
  const items = $derived(draftItems(data.drafts.materials));
  let rows = $state<ReviewRow[]>([]);
  // A reload keeps the edits and positions of drafts still pending.
  $effect(() => {
    const fresh = toDraftRows(data.drafts);
    rows = mergeDraftRows(
      untrack(() => rows),
      fresh,
    );
  });

  let overview = $state<QuickCaptureOverview>(untrack(() => data.overview));
  $effect(() => {
    overview = data.overview;
    pendingDrafts.count = data.overview.pendingDrafts;
  });

  const aiMissing = $derived(!data.aiStatus?.configured);
  const active = $derived(overview.captures.some((c) => c.status === 'queued' || c.status === 'processing'));

  // While captures wait to be read, poll their status; reload the drafts once one finishes.
  $effect(() => {
    if (!active) return;
    const timer = setInterval(async () => {
      try {
        const next = await api.captureInbox();
        const finished = next.captures.some((capture) => {
          const before = overview.captures.find((c) => c.id === capture.id);
          return before && before.status !== capture.status;
        });
        overview = next;
        pendingDrafts.count = next.pendingDrafts;
        if (finished) await invalidate('wm:capture-drafts');
      } catch (error) {
        toasts.error(error, 'Could not refresh the capture status');
      }
    }, POLL_MS);
    return () => clearInterval(timer);
  });

  // Captures sent from the phone while this tab was in the background.
  $effect(() => {
    const reload = () => void invalidate('wm:capture-drafts');
    window.addEventListener('focus', reload);
    return () => window.removeEventListener('focus', reload);
  });

  function approve(batchId: string, chosen: ReviewRow[], inputs: ReviewRow['tx'][]) {
    return api.approveDrafts(
      batchId,
      chosen.map((row, index) => ({ id: row.suggestion.id, transaction: inputs[index] })),
    );
  }

  async function reject(chosen: ReviewRow[]) {
    await api.rejectDrafts(chosen.map((row) => row.suggestion.id));
  }

  // Fallback upload: what the phone Shortcut sends, from the photo library.
  let picker = $state<HTMLInputElement>();
  let files = $state<File[]>([]);
  let note = $state('');
  let sending = $state(false);
  const canSend = $derived(!sending && (files.length > 0 || note.trim() !== ''));

  async function send() {
    sending = true;
    let sent = 0;
    let repeated = 0;
    try {
      const maxBytes = data.aiStatus?.capture?.maxItemBytes ?? DEFAULT_ITEM_BYTES;
      const images = files.length ? files : [undefined];
      for (const file of images) {
        const blob = file ? (await prepareImage(file, maxBytes)).blob : undefined;
        const receipt = await api.submitCapture(blob, note);
        if (receipt.duplicate) repeated++;
        else sent++;
      }
      files = [];
      note = '';
    } catch (error) {
      toasts.error(error, sent ? `${sent} sent, then the upload failed` : 'Upload failed');
    } finally {
      sending = false;
      if (sent || repeated) {
        toasts.push('success', `${sent} queued${repeated ? `, ${repeated} already sent before` : ''}.`);
        await invalidate('wm:capture-drafts');
      }
    }
  }

  let busyCapture = $state<string | null>(null);

  async function act(capture: QuickCapture, action: 'retry' | 'discard') {
    busyCapture = capture.id;
    try {
      if (action === 'retry') await api.retryCapture(capture.id);
      else await api.discardCapture(capture.id);
      await invalidate('wm:capture-drafts');
    } catch (error) {
      toasts.error(error, action === 'retry' ? 'Retry failed' : 'Discard failed');
    } finally {
      busyCapture = null;
    }
  }

  function statusTone(capture: QuickCapture): string {
    if (capture.status === 'failed') return 'text-rose-600 dark:text-rose-400';
    if (capture.status === 'done') return capture.newDrafts ? 'text-emerald-600 dark:text-emerald-400' : 'muted';
    if (capture.attempts > 0) return 'text-amber-600 dark:text-amber-400';
    return 'text-sky-600 dark:text-sky-400';
  }
</script>

<PageHeader title="Drafts" subtitle="Transactions read from screenshots sent by your phone, waiting for your decision">
  {#snippet actions()}
    <a class="btn-ghost" href="{base}/transactions"><Icon name="chevronLeft" size={16} /> Transactions</a>
  {/snippet}
</PageHeader>

{#if aiMissing}
  <Card class="mb-4">
    <p class="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300">
      <Icon name="alert" size={16} />
      AI is not configured on the engine: captures are stored but cannot be read until it is. See
      <a class="underline" href="{base}/settings">Settings</a>.
    </p>
  </Card>
{/if}

<CaptureReview
  bind:rows
  {result}
  {items}
  accounts={data.accounts}
  categories={data.categories}
  tags={data.tags}
  write={approve}
  {reject}
  verb="Approve"
  emptyText="No drafts waiting. Send a screenshot from your phone, or upload one below."
/>

<Card
  title="Captures"
  subtitle="The latest screenshots and notes sent to the engine. Failed reads are retried automatically, then kept until you retry or discard them."
  class="mt-4"
>
  {#if overview.captures.length}
    <ul class="divide-y divide-slate-900/10 text-sm dark:divide-white/10">
      {#each overview.captures as capture (capture.id)}
        <li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
          <span class="muted w-32 shrink-0 text-xs tabular-nums">{formatDateTime(capture.createdAt)}</span>
          <span class="flex min-w-0 flex-1 items-center gap-2">
            {#if capture.hasImage}<Icon name="camera" size={14} class="muted shrink-0" />{/if}
            {#if capture.note}<span class="truncate" title={capture.note}>“{capture.note}”</span>{/if}
          </span>
          <span class="text-xs {statusTone(capture)}">{captureStatus(capture)}</span>
          <span class="flex gap-1">
            {#if capture.status === 'failed'}
              <button
                type="button"
                class="btn-ghost"
                disabled={busyCapture === capture.id}
                onclick={() => act(capture, 'retry')}><Icon name="refresh" size={14} /> Retry</button
              >
            {/if}
            {#if capture.status !== 'processing' && !(capture.status === 'done' && capture.newDrafts > 0)}
              <button
                type="button"
                class="btn-ghost"
                aria-label="Discard capture"
                title="Discard"
                disabled={busyCapture === capture.id}
                onclick={() => act(capture, 'discard')}><Icon name="trash" size={14} /></button
              >
            {/if}
          </span>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted text-sm">Nothing sent yet. Set up the iPhone Shortcut to send screenshots in one tap.</p>
  {/if}
</Card>

<Card title="Upload" subtitle="Send screenshots from this device the same way the phone Shortcut does." class="mt-4">
  <div class="flex flex-wrap items-center gap-2">
    <button type="button" class="btn-ghost" disabled={sending} onclick={() => picker?.click()}>
      <Icon name="plus" size={16} /> Choose screenshots
    </button>
    <span class="muted text-xs">
      {files.length ? `${files.length} selected: ${files.map((file) => file.name).join(', ')}` : 'None selected'}
    </span>
    <input
      bind:this={picker}
      type="file"
      accept="image/*,.heic,.heif"
      multiple
      class="hidden"
      onchange={(event) => {
        const input = event.currentTarget;
        files = [...(input.files ?? [])].filter(isImageFile);
        input.value = '';
      }}
    />
  </div>
  <label class="label mt-3" for="capture-note">Note (optional, added to each draft's memo)</label>
  <textarea
    id="capture-note"
    class="field min-h-16 text-sm"
    placeholder="e.g. team lunch, split with An"
    bind:value={note}
    disabled={sending}
  ></textarea>
  <div class="mt-3 flex justify-end">
    <button type="button" class="btn-primary" disabled={!canSend} onclick={send}>
      <Icon name="send" size={16} />
      {sending ? 'Sending…' : 'Send'}
    </button>
  </div>
</Card>
