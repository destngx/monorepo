<script lang="ts">
  import type { BudgetLine } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
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
</script>

<Card title="Expense categories" subtitle="Click a category for detail" class="mt-4">
  {#if lines.length}
    <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
      {#each lines as line (line.category)}
        <li class="flex items-center gap-3 py-3">
          <button
            type="button"
            class="row-link -m-1.5 min-w-0 flex-1 rounded-lg p-1.5 text-left"
            onclick={() => onselect(line)}
          >
            <div class="mb-1.5 flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="flex items-center gap-2 font-medium">
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
              <p class="mb-1 flex items-center gap-1 text-xs {STATUS.serious.ink}">
                <Icon name="clock" size={12} />Runs out on {formatDate(line.burnOutDate)} at the current pace
              </p>
            {/if}
            <ProgressBar
              percent={line.unbudgeted ? 100 : line.percentUsed}
              color={statusBar[line.status]}
              label={line.category}
            />
          </button>
          <button
            type="button"
            class="btn-ghost px-2"
            aria-label="Edit limit for {line.category}"
            title="Edit limit"
            onclick={() => onedit(line)}><Icon name="pencil" size={15} /></button
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
