<script lang="ts">
  import type { Goal } from '$lib/api/types';
  import Ring from '$lib/charts/Ring.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatPercent } from '$lib/format';

  let { data } = $props();

  const status: Record<Goal['status'], { tone: 'green' | 'amber' | 'red'; label: string; color: string }> = {
    ON_TRACK: { tone: 'green', label: 'On track', color: '#10b981' },
    AT_RISK: { tone: 'amber', label: 'At risk', color: '#f59e0b' },
    OFF_TRACK: { tone: 'red', label: 'Off track', color: '#f43f5e' },
  };
  const percent = (goal: Goal) => (goal.targetAmount ? (goal.currentAmount / goal.targetAmount) * 100 : 0);
</script>

<PageHeader title="Goals" subtitle="Progress toward savings targets" />

{#if data.goals.length}
  <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
    {#each data.goals as goal (goal.id)}
      <Card>
        <div class="flex items-center gap-5">
          <Ring
            percent={percent(goal)}
            color={status[goal.status].color}
            size={112}
            thickness={10}
            label="{goal.name} progress"
          >
            <span class="text-2xl" aria-hidden="true">{goal.emoji}</span>
            <span class="tabular text-xs font-medium">{formatPercent(percent(goal), 0)}</span>
          </Ring>
          <div class="min-w-0 flex-1">
            <p class="truncate font-semibold">{goal.name}</p>
            <p class="mt-1 text-sm"><Money value={goal.currentAmount} /></p>
            <p class="muted text-xs">of <Money value={goal.targetAmount} /></p>
            <div class="mt-2"><Badge tone={status[goal.status].tone}>{status[goal.status].label}</Badge></div>
          </div>
        </div>
      </Card>
    {/each}
  </div>
  <p class="muted mt-4 text-xs">
    Status is based on funding (under 50% is at risk). Pace-based status and projections need deadlines and contribution
    history in the Goals sheet (WM-GOAL-3).
  </p>
{:else}
  <Card><EmptyState title="No goals yet" detail="Add goals to the Goals sheet." /></Card>
{/if}
