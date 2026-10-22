<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { ProgramComparisonReport } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';

  // WM-CARD-14: the last year of card spend replayed under every programme.
  let { class: className = '' }: { class?: string } = $props();
  let busy = $state(false);
  let error = $state<string | null>(null);
  let report = $state<ProgramComparisonReport | null>(null);

  async function compare() {
    busy = true;
    error = null;
    try {
      report = await api.comparePrograms();
    } catch (failure) {
      report = null;
      error = failure instanceof Error ? `Comparison failed: ${failure.message}` : 'Comparison failed.';
    } finally {
      busy = false;
    }
  }
</script>

<Card
  title="Compare programmes"
  subtitle="Your card payments of the last 12 months under each programme's current terms, net of the annual fee"
  class={className}
>
  {#snippet actions()}
    <Button icon="refresh" {busy} busyText="Comparing…" onclick={compare}>{report ? 'Recompare' : 'Compare'}</Button>
  {/snippet}
  {#if error}<p role="alert" class="text-sm text-orange-600 dark:text-orange-400">{error}</p>{/if}
  {#if report}
    <p class="muted mb-3 text-xs">
      {report.payments} payments from {formatDate(report.from)} to {formatDate(report.to)}. Estimated parts rest on
      assumed or unknown MCCs.
    </p>
    {#if report.programs.length === 0}
      <p class="muted text-sm">No programmes in the Programs tab.</p>
    {:else}
      <div class="-mx-5 overflow-x-auto">
        <table class="w-full text-sm">
          <thead
            ><tr class="muted text-left text-xs uppercase"
              ><th class="px-5 py-2">Programme</th><th class="px-5 py-2 text-right">Cashback</th><th
                class="px-5 py-2 text-right">Confirmed</th
              ><th class="px-5 py-2 text-right">Annual fee</th><th class="px-5 py-2 text-right">Net</th></tr
            ></thead
          >
          <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
            {#each report.programs as program (program.program)}
              <tr>
                <td class="px-5 py-2"
                  >{program.program} <span class="muted text-xs">{program.bank}</span>
                  {#if program.held}<Badge tone="blue">held</Badge>{/if}</td
                >
                <td class="px-5 py-2 text-right"><Money value={program.cashback.total} /></td>
                <td class="px-5 py-2 text-right"><Money value={program.cashback.confirmed} /></td>
                <td class="px-5 py-2 text-right"><Money value={program.annualFee} /></td>
                <td class="px-5 py-2 text-right"><Money value={program.net} tone signed /></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {:else if !error}
    <p class="muted text-sm">Run the comparison to see which programme suits how you spend.</p>
  {/if}
</Card>
