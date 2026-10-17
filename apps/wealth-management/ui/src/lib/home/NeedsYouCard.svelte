<script lang="ts">
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { base } from '$app/paths';
  import type { NeedsYou } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import { formatDate } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { inboxHeadline, itemAmount, KIND_ICON, NEEDS_YOU_VISIBLE } from './needsYou';

  // ROADMAP R5: the home page leads with what needs the user, ranked by the
  // engine. Each headline opens the page where it is acted on.
  let { inbox }: { inbox: NeedsYou } = $props();

  const headline = $derived(inboxHeadline(inbox));
  let showAll = $state(false);
  const visible = $derived(showAll ? inbox.items : inbox.items.slice(0, NEEDS_YOU_VISIBLE));
</script>

<PageHeadline question="What needs me?" tone={headline.tone} title={headline.title} id="needs-you-headline">
  <p data-testid="needs-you-text">{headline.text}</p>

  {#if inbox.items.length}
    <ul class="mt-3 divide-y divide-slate-900/5 dark:divide-white/5" aria-label="Needs you">
      {#each visible as item (item.id)}
        {@const amount = itemAmount(item)}
        <li data-kind={item.kind} class="flex items-start gap-3 py-2.5">
          <span class="mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg {STATUS[item.status].wash}"
            ><Icon name={KIND_ICON[item.kind]} size={16} /></span
          >
          <div class="min-w-0 flex-1">
            <a href="{base}{item.route}" class="font-medium text-indigo-700 hover:underline dark:text-indigo-300"
              ><MaskedText text={item.headline} inLink /></a
            >
            {#if item.detail}<p class="muted text-xs"><MaskedText text={item.detail} /></p>{/if}
          </div>
          <div class="flex shrink-0 flex-col items-end gap-1 text-right">
            <Badge status={item.status} />
            {#if amount}
              <span class="text-xs whitespace-nowrap"
                ><Money value={amount.value} />{#if amount.unit}
                  <span class="muted">{amount.unit}</span>{/if}</span
              >
            {/if}
            {#if item.dueDate}<span class="muted text-xs">{formatDate(item.dueDate)}</span>{/if}
          </div>
        </li>
      {/each}
    </ul>
    {#if inbox.items.length > NEEDS_YOU_VISIBLE}
      <button type="button" class="btn-ghost mt-2 px-2 text-xs" onclick={() => (showAll = !showAll)}>
        {showAll ? 'Show fewer' : `Show all ${inbox.items.length}`}
      </button>
    {/if}
  {/if}
</PageHeadline>
