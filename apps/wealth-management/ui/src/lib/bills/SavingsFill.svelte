<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { FillResult, IdleCash } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import { Button } from '@libs/ui-kit';
  import { place, rateLabel, sameAccount } from './idleCash';

  // Describe it for the savings settings (WM-FILL-15, account-savings): AI
  // reads which product an account holds and whether it is emergency money;
  // the change shows here and is saved only by Apply, through the same
  // endpoints as the switches below (D34, D41).
  let { idle }: { idle: IdleCash } = $props();

  interface Proposal {
    account: string;
    product?: string;
    emergency?: boolean;
  }

  let proposal = $state<Proposal | null>(null);
  let problem = $state<string | null>(null);

  const names = $derived([...idle.accounts, ...idle.emergency.accounts].map((account) => account.name));
  const products = $derived(idle.options.filter((option) => option.access === 'flexible'));
  const isEmergency = (name: string) => idle.emergency.accounts.some((account) => sameAccount(account.name, name));
  const linked = (name: string) =>
    idle.emergency.settings.accountProducts.accounts.find((link) => sameAccount(link.account, name))?.product ?? '';
  const productName = (id: string) => {
    const product = products.find((candidate) => candidate.productId === id);
    return product ? `${place(product)}, ${rateLabel(product.rate)}` : id;
  };

  const changes = $derived.by(() => {
    if (!proposal) return [];
    const list: string[] = [];
    if (proposal.product !== undefined && proposal.product !== linked(proposal.account)) {
      list.push(proposal.product ? `Link to ${productName(proposal.product)}` : 'Unlink its product');
    }
    if (proposal.emergency !== undefined && proposal.emergency !== isEmergency(proposal.account)) {
      list.push(proposal.emergency ? 'Mark as emergency money' : 'Mark as an everyday account');
    }
    return list;
  });

  function fill(result: FillResult) {
    problem = null;
    proposal = null;
    const account = typeof result.values.account === 'string' ? result.values.account : '';
    if (!account) return;
    const name = names.find((candidate) => sameAccount(candidate, account));
    if (!name) {
      problem = `${account} is not one of the ready accounts here.`;
      return;
    }
    proposal = {
      account: name,
      product: typeof result.values.product === 'string' ? result.values.product : undefined,
      emergency: typeof result.values.emergency === 'boolean' ? result.values.emergency : undefined,
    };
  }

  async function apply() {
    if (!proposal) return;
    const { account, product, emergency } = proposal;
    if (product !== undefined && product !== linked(account)) {
      await api.setAccountProduct({ account, product, etag: idle.emergency.settings.accountProducts.etag });
    }
    if (emergency !== undefined && emergency !== isEmergency(account)) {
      await api.setEmergencyAccount({ account, emergency, etag: idle.emergency.settings.etag });
    }
    proposal = null;
    await invalidateAll();
  }
</script>

<section class="sm:col-span-2" aria-label="Describe the savings settings">
  <FillBox form="account-savings" onfill={fill} />
  {#if problem}
    <p class="mt-2 text-xs text-amber-700 dark:text-amber-300" role="alert">{problem}</p>
  {:else if proposal}
    <div class="mt-2 rounded-sm border border-edge p-3 text-sm">
      <p class="font-medium">{proposal.account}</p>
      {#if changes.length}
        <ul class="mt-1 list-disc pl-4 text-xs">
          {#each changes as change (change)}<li>{change}</li>{/each}
        </ul>
        <div class="mt-2 flex gap-2">
          <Button
            size="sm"
            variant="primary"
            busyText="Saving…"
            success="{proposal.account} updated."
            errorContext="Could not update {proposal.account}"
            onclick={apply}>Apply</Button
          >
          <Button size="sm" onclick={() => (proposal = null)}>Dismiss</Button>
        </div>
      {:else}
        <p class="muted mt-1 text-xs">Nothing to change: the settings already say so.</p>
      {/if}
    </div>
  {/if}
</section>
