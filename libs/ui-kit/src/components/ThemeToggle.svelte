<script lang="ts">
  import type { Theme, ThemeSetting } from '../state/theme.svelte';
  import Icon, { type IconName } from './Icon.svelte';

  // Cycles the app's theme setting: system, light, dark.
  let { theme }: { theme: ThemeSetting } = $props();

  const icons: Record<Theme, IconName> = { system: 'monitor', light: 'sun', dark: 'moon' };
  const names: Record<Theme, string> = { system: 'System theme', light: 'Light theme', dark: 'Dark theme' };
  const label = $derived(`${names[theme.value]}; click to change`);
</script>

<button type="button" class="icon-btn size-9" aria-label={label} title={label} onclick={() => theme.cycle()}>
  <Icon name={icons[theme.value]} size={18} />
</button>
