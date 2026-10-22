<script lang="ts">
  import type { CaptureLink } from '$lib/api/types';
  import { session } from '$lib/state/session.svelte';
  import CopyField from './CopyField.svelte';
  import Modal from './Modal.svelte';
  import Spinner from './Spinner.svelte';

  // The Mac shows a QR code; the iPhone's camera opens it and the page there
  // takes photos straight into this inbox (HW-IMP-9), or into one unfiled
  // item when itemId is set (HW-IMP-10). While open, it counts what arrives
  // from the phone.
  let { open, onclose, itemId }: { open: boolean; onclose: () => void; itemId?: string } = $props();

  let link = $state<CaptureLink | null>(null);
  let problem = $state('');
  let received = $state(0);

  async function start() {
    link = null;
    problem = '';
    try {
      link = await session.api.capture(location.origin, itemId);
    } catch (error) {
      problem = error instanceof Error ? error.message : String(error);
    }
  }

  $effect(() => {
    if (!open) return;
    received = 0;
    void start();
    let known: Set<string> | null = null;
    const poll = async () => {
      try {
        // An item counts its new files; the inbox counts new phone items.
        const ids = itemId
          ? (await session.api.item(itemId)).item.files.map((f) => f.id)
          : (await session.api.inbox()).filter((i) => i.source === 'phone').map((i) => i.id);
        if (!known) known = new Set(ids);
        else received = ids.filter((id) => !known!.has(id)).length;
      } catch {
        // The count is a nicety; the inbox page shows the items either way.
      }
    };
    void poll();
    const timer = setInterval(() => void poll(), 3000);
    return () => clearInterval(timer);
  });

  const expires = $derived(
    link ? new Date(link.expiresAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '',
  );
</script>

<Modal {open} title={itemId ? 'Add photos with the iPhone' : 'Take photos with the iPhone'} {onclose}>
  {#if problem}
    <p class="text-sm text-danger-600 dark:text-danger-400">{problem}</p>
    <p class="muted mt-2 text-xs">
      The iPhone opens this Mac over Tailscale. Open the app here from the Mac's Tailscale address, or ask an engine
      admin to set Host in Engine settings.
    </p>
    <button class="btn-ghost mt-4" type="button" onclick={() => void start()}>Try again</button>
  {:else if !link}
    <div class="grid place-items-center py-10"><Spinner /></div>
  {:else}
    <div class="flex flex-col items-center gap-3 text-center">
      <img src={link.qr} alt="QR code for the phone camera page" class="size-64 rounded-xl bg-ink-50 p-2" />
      <p class="text-sm">Open the iPhone's Camera app and point it at the code. Take each page, then press Send.</p>
      <p class="muted text-xs">
        The link works until {expires} and can only add to {itemId ? 'this item' : 'this inbox'}. The iPhone needs
        Tailscale on.
      </p>
      {#if received}
        <p class="text-sm font-medium text-accent-700 dark:text-accent-300" role="status">
          {received}
          {itemId ? (received === 1 ? 'page' : 'pages') : received === 1 ? 'item' : 'items'} received from the phone
        </p>
      {/if}
      <details class="w-full text-left">
        <summary class="muted cursor-pointer text-xs">Link instead of a code</summary>
        <div class="mt-2"><CopyField label="Phone camera link" value={link.url} /></div>
      </details>
      <button class="btn-ghost" type="button" onclick={() => void start()}>Make a new code</button>
    </div>
  {/if}
</Modal>
