<script lang="ts">
  import Button from '$lib/components/Button.svelte';
  import type { SubmitHandler } from '$lib/accounts/handlers';
  import type { Category } from '$lib/api/types';
  import { groupCategories } from '$lib/budget/natures';
  import Modal from '$lib/components/Modal.svelte';
  import { KIND_LABELS, type RecurringForm } from './bills';

  interface Props {
    open: boolean;
    title: string;
    form: RecurringForm;
    /** The Categories tab, for the Category list; empty while it loads. */
    categories: Category[];
    categoriesLoading?: boolean;
    categoriesError?: string | null;
    busy: boolean;
    /** Why the last submit was refused, shown above the buttons. */
    error?: string | null;
    submitLabel: string;
    onclose: () => void;
    onsubmit: SubmitHandler;
  }

  let {
    open,
    title,
    form = $bindable(),
    categories,
    categoriesLoading = false,
    categoriesError = null,
    busy,
    error = null,
    submitLabel,
    onclose,
    onsubmit,
  }: Props = $props();

  const isCard = $derived(form.kind === 'card');
  const categoryGroups = $derived(groupCategories(categories));
  // A saved or suggested category missing from the list (renamed, or the list
  // has not loaded) stays a choice, so opening the form does not clear it.
  const unlistedCategory = $derived(
    form.category && !categories.some((category) => category.name === form.category) ? form.category : null,
  );
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
</script>

<Modal {open} {title} {onclose} {busy}>
  <form id="recurring-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" {onsubmit}>
    <div class="sm:col-span-2">
      <label class="label" for="recurring-name">Name *</label>
      <input id="recurring-name" class="field" bind:value={form.name} required />
    </div>
    <div>
      <label class="label" for="recurring-kind">Kind</label>
      <select id="recurring-kind" class="field" bind:value={form.kind}>
        {#each Object.entries(KIND_LABELS) as [value, label] (value)}
          <option {value}>{label}</option>
        {/each}
      </select>
    </div>
    {#if isCard}
      <div>
        <label class="label" for="recurring-account">Card account *</label>
        <input
          id="recurring-account"
          class="field"
          bind:value={form.account}
          aria-describedby="recurring-card-hint"
          required
        />
      </div>
      <p id="recurring-card-hint" class="muted text-xs sm:col-span-2">
        Card payments come from the card statement and are due by the 29th (28th in February). This row only renames the
        payment, adds a note, or hides it when paused.
      </p>
    {:else}
      <div>
        <label class="label" for="recurring-amount">Amount (VND) *</label>
        <input id="recurring-amount" type="number" min="1" step="1" class="field" bind:value={form.amount} required />
      </div>
      <div>
        <label class="label" for="recurring-frequency">How often</label>
        <select id="recurring-frequency" class="field" bind:value={form.frequency}>
          <option value="monthly">Monthly</option>
          <option value="quarterly">Quarterly</option>
          <option value="yearly">Yearly</option>
        </select>
      </div>
      <div>
        <label class="label" for="recurring-day">Day of month *</label>
        <input
          id="recurring-day"
          type="number"
          min="1"
          max="31"
          step="1"
          class="field"
          bind:value={form.day}
          aria-describedby="recurring-day-hint"
          required
        />
        <p id="recurring-day-hint" class="muted mt-1 text-xs">31 means the last day in shorter months.</p>
      </div>
      {#if form.frequency !== 'monthly'}
        <div>
          <label class="label" for="recurring-month"
            >{form.frequency === 'quarterly' ? 'First month *' : 'Month *'}</label
          >
          <select id="recurring-month" class="field" bind:value={form.month}>
            {#each months as label, index (label)}
              <option value={index + 1}>{label}</option>
            {/each}
          </select>
        </div>
      {/if}
      <div class="sm:col-span-2">
        <label class="label" for="recurring-match">Payee contains</label>
        <input id="recurring-match" class="field" bind:value={form.match} aria-describedby="recurring-match-hint" />
        <p id="recurring-match-hint" class="muted mt-1 text-xs">
          Whole words, case and accents ignored. Without it, the category and an amount within 15% mark a payment.
        </p>
      </div>
      <div>
        <label class="label" for="recurring-category">Category</label>
        <select
          id="recurring-category"
          class="field"
          bind:value={form.category}
          aria-describedby={categoriesLoading || categoriesError ? 'recurring-category-hint' : undefined}
        >
          <option value="">No category</option>
          {#if unlistedCategory}
            <option value={unlistedCategory}>{unlistedCategory} (not in Categories)</option>
          {/if}
          {#each categoryGroups as group (group.label)}
            <optgroup label={group.label}>
              {#each group.categories as category (category.name)}
                <option value={category.name}>{category.name}</option>
              {/each}
            </optgroup>
          {/each}
        </select>
        {#if categoriesError}
          <p id="recurring-category-hint" class="muted mt-1 text-xs">
            Could not load the categories: {categoriesError}
          </p>
        {:else if categoriesLoading}
          <p id="recurring-category-hint" class="muted mt-1 text-xs">Loading the categories…</p>
        {/if}
      </div>
      <div>
        <label class="label" for="recurring-account">Account</label>
        <input id="recurring-account" class="field" bind:value={form.account} />
      </div>
    {/if}
    <div>
      <label class="label" for="recurring-status">Status</label>
      <select id="recurring-status" class="field" bind:value={form.status}>
        <option value="active">Active</option>
        <option value="paused">Paused</option>
      </select>
    </div>
    <div>
      <label class="label" for="recurring-note">Note</label>
      <input id="recurring-note" class="field" bind:value={form.note} />
    </div>
    {#if error}<p role="alert" class="text-sm text-orange-600 sm:col-span-2 dark:text-orange-400">{error}</p>{/if}
  </form>
  {#snippet footer()}
    <Button disabled={busy} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="recurring-form" variant="primary" {busy} busyText="Saving…">{submitLabel}</Button>
  {/snippet}
</Modal>
