<script lang="ts">
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import { Icon } from '@libs/ui-kit';
  import { todayISO } from '$lib/format';
  import { entryReminder } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { hasEntryToday, parseReminderHour, reminderDue } from './reminder';

  // Daily entry reminder banner (WM-TX-9), checked once the sheet is reachable.
  let { ready }: { ready: boolean } = $props();

  let show = $state(false);
  let checked = false;

  $effect(() => {
    const due = reminderDue({
      enabled: entryReminder.enabled.value === 'true',
      hour: parseReminderHour(entryReminder.hour.value),
      now: new Date(),
      dismissedOn: entryReminder.dismissedOn.value,
    });
    if (!ready || !due || checked) return;
    checked = true;
    api
      .transactions()
      .then((transactions) => (show = !hasEntryToday(transactions, new Date())))
      .catch((error: unknown) => toasts.error(error, 'Could not check today’s entries'));
  });

  function dismiss() {
    entryReminder.dismissedOn.value = todayISO();
    show = false;
  }
</script>

{#if show}
  <div role="status" class="panel mb-6 flex items-center gap-3 border border-sky-500/40 p-4 text-sm">
    <Icon name="clock" class="text-sky-600 dark:text-sky-400" />
    <p class="flex-1">
      Nothing recorded today yet.
      <a class="text-accent-600 hover:underline dark:text-accent-400" href="{base}/transactions?new=1" onclick={dismiss}
        >Add a transaction</a
      >
    </p>
    <button type="button" class="opacity-60 hover:opacity-100" aria-label="Dismiss for today" onclick={dismiss}>
      <Icon name="x" size={16} />
    </button>
  </div>
{/if}
