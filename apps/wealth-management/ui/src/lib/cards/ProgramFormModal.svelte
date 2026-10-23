<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { CapPeriod, CardConfig, FillResult, FillValues, ProgramRuleInput } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill, asNumber, asText, notesFor } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { todayISO } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';

  // Adds a cashback programme version with its rules (WM-FILL-13). AI reads
  // the bank's terms (a page or a screenshot) into one row per rule.
  interface Props {
    open: boolean;
    onclose: () => void;
  }

  let { open, onclose }: Props = $props();

  interface Form {
    program: string;
    bank: string;
    sharedAccount: string;
    from: string;
    annualFee: number | null;
    cap: number | null;
    capPeriod: CapPeriod;
  }

  interface Rule {
    id: number;
    rule: string;
    mccs: string;
    rate: number | null;
    cap: number | null;
    minTransaction: number | null;
    minCycleSpend: number | null;
    fallback: boolean;
    exclude: boolean;
    capGroup: string;
    notes: string[];
  }

  const empty = (): Form => ({
    program: '',
    bank: '',
    sharedAccount: '',
    from: todayISO(),
    annualFee: null,
    cap: null,
    capPeriod: 'cycle',
  });
  let form = $state<Form>(empty());
  let rules = $state<Rule[]>([]);
  let config = $state<CardConfig | null>(null);
  let accounts = $state<string[]>([]);
  let saving = $state(false);
  let nextId = 0;
  const marks = new FillMarks();

  $effect(() => {
    if (!open) {
      form = empty();
      rules = [];
      marks.clear();
      return;
    }
    Promise.all([api.cardPrograms(), api.accounts()]).then(
      ([loadedConfig, loadedAccounts]) => {
        config = loadedConfig;
        accounts = loadedAccounts.map((account) => account.name);
      },
      (error: unknown) => toasts.error(error, 'Could not read the card tabs'),
    );
  });

  const fold = (text: string) => text.trim().toLowerCase();
  const banks = $derived(config?.banks ?? []);
  const knownBank = $derived(banks.find((bank) => fold(bank.name) === fold(form.bank)));
  const ready = $derived(rules.filter((rule) => rule.rule.trim() && rule.rate !== null && rule.rate >= 0));
  const valid = $derived(
    form.program.trim() !== '' &&
      form.bank.trim() !== '' &&
      form.from !== '' &&
      ready.length > 0 &&
      ready.length === rules.length,
  );

  const blank = (): Rule => ({
    id: nextId++,
    rule: '',
    mccs: '',
    rate: null,
    cap: null,
    minTransaction: null,
    minCycleSpend: null,
    fallback: false,
    exclude: false,
    capGroup: '',
    notes: [],
  });

  function ruleOf(row: FillValues, index: number, result: FillResult): Rule {
    return {
      ...blank(),
      rule: asText(row.rule),
      mccs: asText(row.mccs),
      rate: asNumber(row.rate),
      cap: asNumber(row.cap),
      minTransaction: asNumber(row.minTransaction),
      minCycleSpend: asNumber(row.minCycleSpend),
      fallback: row.fallback === true,
      exclude: row.exclude === true,
      capGroup: asText(row.capGroup),
      notes: ['rule', 'mccs', 'rate', 'cap', 'minTransaction', 'minCycleSpend', 'capGroup']
        .flatMap((field) => notesFor(result.notes, field, index))
        .map((note) => note.message),
    };
  }

  function fill(result: FillResult) {
    form = applyFill(form, result.values, {
      program: asText,
      bank: asText,
      sharedAccount: asText,
      from: asText,
      annualFee: asNumber,
      cap: asNumber,
      capPeriod: (value) => (value === 'month' ? 'month' : 'cycle'),
    });
    if (result.rows?.length) rules = result.rows.map((row, index) => ruleOf(row, index, result));
    marks.take(result);
  }

  const optional = (value: number | null) => (value ? value : undefined);

  async function save() {
    saving = true;
    try {
      const program = await api.createCardProgram({
        program: form.program.trim(),
        bank: form.bank.trim(),
        sharedAccount: knownBank ? undefined : form.sharedAccount.trim() || undefined,
        from: form.from,
        annualFee: optional(form.annualFee),
        cap: optional(form.cap),
        capPeriod: form.capPeriod,
        rules: rules.map((rule): ProgramRuleInput => ({
          rule: rule.rule.trim(),
          mccs: rule.mccs.trim() || undefined,
          rate: rule.rate!,
          cap: optional(rule.cap),
          minTransaction: optional(rule.minTransaction),
          minCycleSpend: optional(rule.minCycleSpend),
          fallback: rule.fallback || undefined,
          exclude: rule.exclude || undefined,
          capGroup: rule.capGroup.trim() || undefined,
        })),
      });
      toasts.success(`Saved the ${program.name} terms from ${form.from}.`);
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not save the programme');
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="Add programme terms" {onclose} busy={saving}>
  <FillBox form="cashback-program" rowsLabel="rule" class="mb-4" current={() => ({ ...form })} onfill={fill} />
  <datalist id="program-names">
    {#each config?.programs ?? [] as program (program.name)}<option value={program.name}></option>{/each}
  </datalist>
  <datalist id="program-banks">
    {#each banks as bank (bank.name)}<option value={bank.name}></option>{/each}
  </datalist>
  <datalist id="program-accounts">
    {#each accounts as account (account)}<option value={account}></option>{/each}
  </datalist>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
    <div>
      <label class="label" for="program-name">Programme *<FilledMark show={marks.has('program', form.program)} /></label
      >
      <input id="program-name" class="field" list="program-names" bind:value={form.program} required />
    </div>
    <div>
      <label class="label" for="program-bank">Bank *<FilledMark show={marks.has('bank', form.bank)} /></label>
      <input id="program-bank" class="field" list="program-banks" bind:value={form.bank} required />
    </div>
    {#if form.bank.trim() && !knownBank}
      <div class="sm:col-span-2">
        <label class="label" for="program-shared"
          >Shared account<FilledMark show={marks.has('sharedAccount', form.sharedAccount)} /></label
        >
        <input
          id="program-shared"
          class="field"
          list="program-accounts"
          bind:value={form.sharedAccount}
          aria-describedby="program-shared-hint"
        />
        <p id="program-shared-hint" class="muted mt-1 text-xs">
          {form.bank.trim()} is a new bank: the account its cards bill to. A card needs it.
        </p>
      </div>
    {/if}
    <div>
      <label class="label" for="program-from">Terms from *<FilledMark show={marks.has('from', form.from)} /></label>
      <input id="program-from" type="date" class="field" bind:value={form.from} required />
    </div>
    <div>
      <label class="label" for="program-period"
        >Cap period<FilledMark show={marks.has('capPeriod', form.capPeriod)} /></label
      >
      <select id="program-period" class="field" bind:value={form.capPeriod}>
        <option value="cycle">Each statement cycle</option>
        <option value="month">Each calendar month</option>
      </select>
    </div>
    <div>
      <label class="label" for="program-fee"
        >Annual fee (VND)<FilledMark show={marks.has('annualFee', form.annualFee)} /></label
      >
      <input id="program-fee" type="number" min="0" step="1" class="field" bind:value={form.annualFee} />
    </div>
    <div>
      <label class="label" for="program-cap">Card cap (VND)<FilledMark show={marks.has('cap', form.cap)} /></label>
      <input id="program-cap" type="number" min="0" step="1" class="field" bind:value={form.cap} />
    </div>
  </div>

  <h3 class="mt-5 text-sm font-medium">Rules</h3>
  <ul class="mt-2 space-y-3" aria-label="Rules">
    {#each rules as rule (rule.id)}
      <li class="rounded-sm border border-edge p-3">
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-[1fr_1fr_6rem_8rem_auto] sm:items-end">
          <div>
            <label class="label" for="rule-name-{rule.id}">Rule</label>
            <input id="rule-name-{rule.id}" class="field" bind:value={rule.rule} />
          </div>
          <div>
            <label class="label" for="rule-mccs-{rule.id}">MCCs</label>
            <input id="rule-mccs-{rule.id}" class="field" placeholder="Any" bind:value={rule.mccs} />
          </div>
          <div>
            <label class="label" for="rule-rate-{rule.id}">Rate %</label>
            <input
              id="rule-rate-{rule.id}"
              type="number"
              min="0"
              max="100"
              step="any"
              class="field"
              bind:value={rule.rate}
            />
          </div>
          <div>
            <label class="label" for="rule-cap-{rule.id}">Cap (VND)</label>
            <input id="rule-cap-{rule.id}" type="number" min="0" step="1" class="field" bind:value={rule.cap} />
          </div>
          <Button
            size="icon"
            icon="x"
            label="Drop the {rule.rule || 'new'} rule"
            onclick={() => (rules = rules.filter((other) => other.id !== rule.id))}
          />
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-end">
          <div>
            <label class="label" for="rule-min-{rule.id}">Min transaction</label>
            <input
              id="rule-min-{rule.id}"
              type="number"
              min="0"
              step="1"
              class="field"
              bind:value={rule.minTransaction}
            />
          </div>
          <div>
            <label class="label" for="rule-spend-{rule.id}">Min cycle spend</label>
            <input
              id="rule-spend-{rule.id}"
              type="number"
              min="0"
              step="1"
              class="field"
              bind:value={rule.minCycleSpend}
            />
          </div>
          <div>
            <label class="label" for="rule-group-{rule.id}">Cap group</label>
            <input id="rule-group-{rule.id}" class="field" bind:value={rule.capGroup} />
          </div>
          <div class="flex gap-3 text-sm">
            <label class="flex items-center gap-1.5"
              ><input type="checkbox" bind:checked={rule.fallback} />Fallback</label
            >
            <label class="flex items-center gap-1.5"><input type="checkbox" bind:checked={rule.exclude} />Exclude</label
            >
          </div>
        </div>
        {#each rule.notes as note, index (index)}
          <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{note}</p>
        {/each}
      </li>
    {/each}
  </ul>
  <Button class="mt-3" size="sm" icon="plus" onclick={() => (rules = [...rules, blank()])}>Add a rule</Button>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button variant="primary" busy={saving} busyText="Saving…" disabled={!valid} onclick={save}>Save terms</Button>
  {/snippet}
</Modal>
