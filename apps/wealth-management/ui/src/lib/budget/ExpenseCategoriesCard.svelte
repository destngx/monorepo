<script lang="ts">
  import type { BudgetLine } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Card, EmptyState, Icon } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatDate } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { statusBar, statusLabel, statusTone, type LineHandler } from './status';

  interface Props {
    lines: BudgetLine[];
    onselect: LineHandler;
    onedit: LineHandler;
  }

  let { lines, onselect, onedit }: Props = $props();

  // A dense two-column list: the fullest categories first, the rest one
  // click away so a long budget does not push the page down.
  const SHOWN = 8;
  let showAll = $state(false);
  const visible = $derived(showAll ? lines : lines.slice(0, SHOWN));
</script>

<Card title="Expense categories" subtitle="Fullest first. Click a category for detail" class="mt-4">
  {#snippet actions()}
    {#if lines.length > SHOWN}
      <button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (showAll = !showAll)}
        >{showAll ? 'Show fewer' : `Show all ${lines.length}`}</button
      >
    {/if}
  {/snippet}
  {#if lines.length}
    <ul class="grid grid-cols-1 gap-x-6 md:grid-cols-2">
      {#each visible as line (line.category)}
        <li class="flex items-center gap-1 border-b border-stone-900/5 py-1.5 dark:border-white/5">
          <button
            type="button"
            class="row-link -mx-1.5 min-w-0 flex-1 rounded-lg px-1.5 py-1 text-left"
            onclick={() => onselect(line)}
          >
            <div class="mb-1 flex flex-wrap items-center justify-between gap-x-2 text-sm">
              <span class="flex min-w-0 items-center gap-2 font-medium">
                {line.category}
                {#if line.unbudgeted}<Badge tone="violet">No limit</Badge>
                {:else if line.status !== 'healthy'}<Badge status={statusTone[line.status]}
                    >{statusLabel[line.status]}</Badge
                  >{/if}
              </span>
              <span class="muted tabular text-xs"
                ><Money value={line.spent} /> / <Money value={line.limit} /> · <Money value={line.remaining} /> left</span
              >
            </div>
            {#if line.burnOutDate}
              <p class="mb-0.5 flex items-center gap-1 text-xs {STATUS.serious.ink}">
                <Icon name="clock" size={12} />Runs out on {formatDate(line.burnOutDate)} at the current pace
              </p>
            {/if}
            <ProgressBar
              percent={line.unbudgeted ? 100 : line.percentUsed}
              color={statusBar[line.status]}
              label={line.category}
              size="sm"
            />
            {#if line.splits?.length}
              <!-- WM-CAT-3: the share of the month on the category's split tags. -->
              <p class="muted mt-0.5 flex flex-wrap gap-x-3 text-xs" data-testid="splits">
                {#each line.splits as split (split.tag)}
                  <span
                    >of which <span class="font-medium">#{split.tag}</span>
                    <Money value={split.spent} /> ({Math.round(split.percentOfSpent)}%)</span
                  >
                {/each}
              </p>
            {/if}
          </button>
          <button
            type="button"
            class="btn-ghost shrink-0 px-1.5"
            aria-label="Edit limit for {line.category}"
            title="Edit limit"
            onclick={() => onedit(line)}><Icon name="pencil" size={14} /></button
          >
        </li>
      {/each}
    </ul>
  {:else}
    <EmptyState
      title="No expense budget for this month"
      detail="Set limits in the Budget sheet or edit a category limit here."
    />
  {/if}
</Card>
