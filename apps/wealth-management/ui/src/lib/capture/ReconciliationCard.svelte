<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import type { Reconciliation } from './review';

  interface Props {
    reconciliation: Reconciliation[];
  }

  let { reconciliation }: Props = $props();
</script>

<Card
  title="Balance check"
  subtitle="Balances shown in your materials vs the sheet after the selected rows. A difference hints at a missing or duplicated transaction; no adjustment is ever created."
  class="mb-4"
>
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[40rem] text-sm">
      <thead>
        <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
          <th class="px-5 py-2 font-medium">Account</th>
          <th class="px-5 py-2 text-right font-medium">Shown in material</th>
          <th class="px-5 py-2 text-right font-medium">Sheet + selected</th>
          <th class="px-5 py-2 text-right font-medium">Difference</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each reconciliation as line (line.account)}
          <tr>
            <td class="px-5 py-2">
              {line.account}
              <p class="muted text-xs">as of {line.asOf} · {line.materialId}</p>
            </td>
            <td class="px-5 py-2 text-right"><Money value={line.balanceAfter} currency={line.currency} /></td>
            <td class="px-5 py-2 text-right">
              <Money value={line.projected} currency={line.currency} />
              <p class="muted text-xs">
                <Money value={line.sheetBalance} currency={line.currency} /> + <Money
                  value={line.pending}
                  currency={line.currency}
                  signed
                />
              </p>
            </td>
            <td class="px-5 py-2 text-right">
              {#if line.difference === 0}
                <Badge status="good">Matches</Badge>
              {:else}
                <Money value={line.difference} currency={line.currency} signed tone />
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</Card>
