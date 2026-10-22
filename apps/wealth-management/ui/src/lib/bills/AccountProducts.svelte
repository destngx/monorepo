<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { IdleCash } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import { action } from '$lib/state/action.svelte';
  import { place, rateLabel, sameAccount } from './idleCash';

  // Linked wallet products (D41): an account that is a wallet earning balance
  // earns the product's live rate. Each link is saved with the setting's ETag.
  // One change runs at a time, since each change moves the ETag.
  let { idle }: { idle: IdleCash } = $props();
  let pending = $state<string | null>(null);

  const settings = $derived(idle.emergency.settings.accountProducts);
  const products = $derived(idle.options.filter((option) => option.access === 'flexible'));
  const names = $derived([...idle.accounts, ...idle.emergency.accounts].map((account) => account.name));
  const linked = (name: string) => settings.accounts.find((link) => sameAccount(link.account, name))?.product ?? '';
  const orphans = $derived(settings.accounts.filter((link) => !names.some((name) => sameAccount(name, link.account))));

  async function save(account: string, product: string) {
    pending = account;
    try {
      await api.setAccountProduct({ account, product, etag: settings.etag });
      await invalidateAll();
    } finally {
      pending = null;
    }
  }
  const link = action(save, { errorContext: 'Could not link the account' });

  async function change(event: Event & { currentTarget: HTMLSelectElement }, account: string) {
    const select = event.currentTarget;
    const ok = await link.run(account, select.value);
    // A failed save leaves the saved link in place: show it again.
    if (!ok) select.value = linked(account);
  }

  const waits = (account: string) => pending !== null && pending !== account;
</script>

<section aria-labelledby="account-products-title" data-testid="account-products">
  <h3 id="account-products-title" class="text-xs font-semibold tracking-wide uppercase">Linked wallet products</h3>
  <p class="muted text-xs">
    Linked wallet products: an account that is a wallet earning balance earns the product's live rate.
  </p>
  <ul class="mt-1 divide-y divide-slate-900/5 dark:divide-white/5">
    {#each names as name (name)}
      <li class="flex items-center justify-between gap-2 py-1.5">
        <label class="min-w-0 truncate" for="account-product-{name}">{name}</label>
        <select
          id="account-product-{name}"
          class="field w-56"
          value={linked(name)}
          disabled={waits(name)}
          aria-busy={pending === name || undefined}
          onchange={(event) => change(event, name)}
        >
          <option value="">Not linked</option>
          {#each products as product (product.productId)}
            <option value={product.productId}>{place(product)}, {rateLabel(product.rate)}</option>
          {/each}
          {#if linked(name) && !products.some((product) => product.productId === linked(name))}
            <!-- Still saved, but the market data server does not list it today. -->
            <option value={linked(name)}>{linked(name)} (rate unavailable today)</option>
          {/if}
        </select>
      </li>
    {:else}
      <li class="muted py-1.5">No account to link.</li>
    {/each}
  </ul>
  {#if orphans.length}
    <ul class="mt-2 space-y-1 text-xs" aria-label="Linked products to fix">
      {#each orphans as orphan (orphan.account)}
        <li class="flex items-center justify-between gap-2">
          <span class="min-w-0 truncate">“{orphan.account}” is linked to a product but is not an account.</span>
          <Button
            size="sm"
            label="Unlink {orphan.account}"
            busy={pending === orphan.account}
            busyText="Removing…"
            disabled={waits(orphan.account)}
            success="“{orphan.account}” unlinked."
            errorContext="Could not unlink “{orphan.account}”"
            onclick={() => save(orphan.account, '')}>Unlink</Button
          >
        </li>
      {/each}
    </ul>
  {/if}
</section>
