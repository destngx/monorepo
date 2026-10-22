<script lang="ts">
  import type { InvestmentAction, InvestmentCommand } from '$lib/api/types';
  import Icon, { type IconName } from '$lib/components/Icon.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';

  // Executable commands of the actions phase (WM-INV-18). They are advice
  // only: nothing is written to the sheet. Model text may quote amounts, so
  // it goes through MaskedText (WM-UI-2, WM-UI-15).
  let { commands }: { commands: InvestmentCommand[] } = $props();

  const styles: Record<InvestmentAction, { icon: IconName; card: string; tag: string }> = {
    ALLOCATE: {
      icon: 'investments',
      card: 'border-emerald-500/40 bg-emerald-500/5',
      tag: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
    },
    HEDGE: {
      icon: 'shield',
      card: 'border-amber-500/40 bg-amber-500/5',
      tag: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    },
    OBSERVE: {
      icon: 'eye',
      card: 'border-sky-500/40 bg-sky-500/5',
      tag: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
    },
  };
</script>

<ul class="grid grid-cols-1 gap-3 md:grid-cols-2">
  {#each commands as command, i (i)}
    {@const style = styles[command.action] ?? styles.OBSERVE}
    <li class="rounded-xl border p-4 {style.card}">
      <div class="flex flex-wrap items-center gap-2">
        <span class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-semibold {style.tag}">
          <Icon name={style.icon} size={12} />
          {command.action}
        </span>
        <span class="font-medium break-words"><MaskedText text={command.target} /></span>
      </div>
      <p class="mt-2 text-sm text-stone-700 dark:text-stone-300"><MaskedText text={command.rationale} /></p>
    </li>
  {/each}
</ul>
