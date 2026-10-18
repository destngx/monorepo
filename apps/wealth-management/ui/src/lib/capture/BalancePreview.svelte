<script lang="ts">
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import type { BalanceProjection } from './review';

  // Sheet balances before and after the selected rows are written.
  interface Props {
    projections: BalanceProjection[];
    /** Button verb: "Commit" or "Approve". */
    verb: string;
  }

  let { projections, verb }: Props = $props();
</script>

<Card
  title="Balances after {verb.toLowerCase()}"
  subtitle="Each account's sheet balance now and once the selected rows are written."
  class="mb-4"
>
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[36rem] text-sm">
      <thead>
        <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
          <th class="px-5 py-2 font-medium">Account</th>
          <th class="px-5 py-2 text-right font-medium">Now</th>
          <th class="px-5 py-2 text-right font-medium">Change</th>
          <th class="px-5 py-2 text-right font-medium">After</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each projections as line (line.account)}
          <tr>
            <td class="px-5 py-2">
              {line.account}
              <p class="muted text-xs">{line.rows} row{line.rows === 1 ? '' : 's'}</p>
            </td>
            <td class="px-5 py-2 text-right"><Money value={line.current} currency={line.currency} /></td>
            <td class="px-5 py-2 text-right"><Money value={line.change} currency={line.currency} signed tone /></td>
            <td class="px-5 py-2 text-right font-medium"><Money value={line.after} currency={line.currency} /></td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</Card>
