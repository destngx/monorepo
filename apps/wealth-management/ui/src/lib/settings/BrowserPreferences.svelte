<script lang="ts">
  import { Card } from '@libs/ui-kit';
  import { entryReminder, theme, type Theme } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { parseReminderHour } from '$lib/transactions/reminder';

  const themes: [Theme, string][] = [
    ['system', 'Follow the system'],
    ['light', 'Light'],
    ['dark', 'Dark'],
  ];
  const hours = Array.from({ length: 24 }, (_, hour) => String(hour));

  function setReminder(enabled: boolean) {
    entryReminder.enabled.value = enabled ? 'true' : 'false';
    toasts.info(enabled ? 'Daily entry reminder on.' : 'Daily entry reminder off.');
  }

  function setReminderHour(value: string) {
    entryReminder.hour.value = String(parseReminderHour(value));
    toasts.info(`Reminder shows after ${entryReminder.hour.value}:00.`);
  }
</script>

<Card title="This browser" subtitle="Stored only in this browser">
  <div class="-my-3 divide-y divide-stone-900/5 dark:divide-white/5">
    <div class="grid grid-cols-1 gap-x-6 gap-y-1.5 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center">
      <label class="text-sm font-medium" for="theme">Theme</label>
      <select
        id="theme"
        class="field"
        value={theme.value}
        onchange={(event) => (theme.value = event.currentTarget.value as Theme)}
      >
        {#each themes as [value, label] (value)}<option {value}>{label}</option>{/each}
      </select>
    </div>

    <div class="grid grid-cols-1 gap-x-6 gap-y-1.5 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center">
      <div>
        <label class="text-sm font-medium" for="reminder">Daily entry reminder</label>
        <p class="muted text-xs">A banner when nothing is recorded today.</p>
      </div>
      <div class="flex items-center gap-3 text-sm">
        <input
          id="reminder"
          type="checkbox"
          class="size-4 accent-accent-600"
          checked={entryReminder.enabled.value === 'true'}
          onchange={(event) => setReminder(event.currentTarget.checked)}
        />
        <label for="reminder-hour" class="muted">after</label>
        <select
          id="reminder-hour"
          class="field w-24"
          value={entryReminder.hour.value}
          disabled={entryReminder.enabled.value !== 'true'}
          onchange={(event) => setReminderHour(event.currentTarget.value)}
        >
          {#each hours as hour (hour)}<option value={hour}>{hour.padStart(2, '0')}:00</option>{/each}
        </select>
        <span class="muted text-xs">Vietnam time</span>
      </div>
    </div>
  </div>
</Card>
