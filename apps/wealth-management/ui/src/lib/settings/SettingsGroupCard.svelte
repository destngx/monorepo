<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Card } from '@libs/ui-kit';
  import type { EngineSettingsForm } from './engine.svelte';
  import type { GroupedSettings } from './form';
  import { splitAdvanced } from './sections';
  import SettingField from './SettingField.svelte';

  interface Props {
    grouped: GroupedSettings;
    form: EngineSettingsForm;
    actions?: Snippet;
    /** A note above the settings. */
    intro?: Snippet;
    /** Extra content under the settings. */
    children?: Snippet;
  }

  let { grouped, form, actions, intro, children }: Props = $props();

  const split = $derived(splitAdvanced(grouped.settings));
  // Stays open while it holds an unsaved change, so no edit is out of sight.
  const advancedChanged = $derived(split.advanced.some((setting) => setting.key in form.changes));
</script>

<Card title={grouped.group.label} subtitle={grouped.group.description} {actions}>
  {@render intro?.()}
  {#if split.main.length}
    <div class="-my-3 divide-y divide-stone-900/5 dark:divide-white/5">
      {#each split.main as setting (setting.key)}<SettingField {setting} {form} />{/each}
    </div>
  {/if}
  {#if split.advanced.length}
    <details
      class="{split.main.length ? 'mt-3 border-t pt-3' : ''} border-stone-900/5 dark:border-white/5"
      open={advancedChanged}
    >
      <summary
        class="muted cursor-pointer text-xs font-medium select-none hover:text-stone-700 dark:hover:text-stone-200"
        >Advanced ({split.advanced.length})</summary
      >
      <div class="divide-y divide-stone-900/5 dark:divide-white/5">
        {#each split.advanced as setting (setting.key)}<SettingField {setting} {form} />{/each}
      </div>
    </details>
  {/if}
  {@render children?.()}
</Card>
