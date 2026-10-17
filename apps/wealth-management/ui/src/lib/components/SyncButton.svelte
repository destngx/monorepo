<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { sync } from '$lib/state/sync.svelte';
  import Button from './Button.svelte';

  // Ticks once a second so the countdown stays live.
  let now = $state(Date.now());
  $effect(() => {
    const timer = setInterval(() => (now = Date.now()), 1000);
    return () => clearInterval(timer);
  });

  const countdown = $derived.by(() => {
    if (!sync.nextSyncAt) return '';
    const seconds = Math.max(0, Math.round((sync.nextSyncAt - now) / 1000));
    return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
  });
  const syncedAt = $derived(
    sync.lastSynced?.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', timeZone: 'Asia/Ho_Chi_Minh' }),
  );
</script>

<div class="flex items-center gap-2">
  <div class="hidden text-right text-[11px] leading-tight sm:block">
    {#if sync.lastError}
      <p class="text-orange-600 dark:text-orange-400" title={sync.lastError}>Sync failed</p>
    {:else if syncedAt}
      <p class="muted">Synced: {syncedAt}</p>
    {/if}
    {#if countdown}<p class="muted">Next in {countdown}</p>{/if}
  </div>
  <!-- sync.run toasts its own outcome; busy also covers auto-sync -->
  <Button
    icon="refresh"
    busy={sync.syncing}
    label="Sync with Google Sheets"
    onclick={() => sync.run(invalidateAll, { notify: true })}
  >
    <span class="hidden md:inline">{sync.syncing ? 'Syncing…' : 'Sync'}</span>
  </Button>
</div>
