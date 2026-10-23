<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Icon } from '@libs/ui-kit';
  import Money from './Money.svelte';

  // One folded row of a card's details: a small uppercase title with an
  // optional amount or count, closed until clicked. Rows stack inside a
  // `divide-y` list.
  interface Props {
    title: string;
    /** An amount to show beside the title (masked like any amount). */
    money?: number;
    /** A count to show beside the title. */
    count?: number;
    testid?: string;
    children: Snippet;
  }

  let { title, money, count, testid, children }: Props = $props();
  const id = $props.id();
</script>

<details class="group" data-testid={testid}>
  <summary
    class="flex cursor-pointer list-none items-center justify-between gap-2 py-2 text-xs font-semibold tracking-wide uppercase"
  >
    <span id="{id}-title">{title}</span>
    <span class="muted flex items-center gap-2 font-normal tracking-normal normal-case">
      {#if money !== undefined}<Money value={money} />{:else if count !== undefined}{count}{/if}
      <Icon name="chevronDown" size={14} class="transition-transform group-open:rotate-180" />
    </span>
  </summary>
  <section class="pb-2" aria-labelledby="{id}-title">{@render children()}</section>
</details>
