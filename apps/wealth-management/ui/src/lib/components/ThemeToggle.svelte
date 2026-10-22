<script lang="ts">
  import { theme, type Theme } from '$lib/state/settings.svelte';
  import { Icon, type IconName } from '@libs/ui-kit';

  const order: Theme[] = ['system', 'light', 'dark'];
  const icons: Record<Theme, IconName> = { system: 'monitor', light: 'sun', dark: 'moon' };
</script>

<button
  type="button"
  class="btn-ghost px-2.5"
  title="Theme: {theme.value}"
  aria-label="Theme: {theme.value}. Switch theme"
  onclick={() => (theme.value = order[(order.indexOf(theme.value) + 1) % order.length])}
>
  <Icon name={icons[theme.value]} />
</button>
