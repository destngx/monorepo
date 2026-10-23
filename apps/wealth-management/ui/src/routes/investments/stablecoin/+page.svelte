<script lang="ts">
  import { Button, Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import type { FillResult } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { asNumber } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import {
    DEFAULT_SWITCH_INPUT,
    SWITCH_STORAGE_KEY,
    compareStablecoinSwitch,
    parseSwitchInput,
    type FeeKind,
    type SwitchInput,
  } from '$lib/investments/stablecoinSwitch';

  // WM-INV-30 (D43): the sheet's stablecoin yield vs fees tab, as a tool.
  // Nothing loads; the figures stay in this browser between visits.
  const UNIT = 'USD';

  let input = $state<SwitchInput>(parseSwitchInput(globalThis.localStorage?.getItem(SWITCH_STORAGE_KEY)));
  const result = $derived(compareStablecoinSwitch(input));
  const errors = $derived(result.ok ? {} : result.errors);

  $effect(() => {
    globalThis.localStorage?.setItem(SWITCH_STORAGE_KEY, JSON.stringify(input));
  });

  // An emptied number field binds null; keep it as NaN so it is flagged, not saved as 0.
  const numberOf = (value: number | null) => (value === null ? Number.NaN : value);

  function reset() {
    input = structuredClone(DEFAULT_SWITCH_INPUT);
    marks.clear();
  }

  // Describe it (WM-FILL-6): the figures as the engine's stablecoin form
  // names them, flat.
  const marks = new FillMarks();
  const flat = () => ({
    principal: input.principal,
    currentRate: input.currentRate,
    newRate: input.newRate,
    days: input.days,
    entryFeeKind: input.entryFee.kind,
    entryFee: input.entryFee.value,
    exitFeeKind: input.exitFee.kind,
    exitFee: input.exitFee.value,
  });

  function fill(result: FillResult) {
    const values = result.values;
    const number = (key: string, fallback: number) => asNumber(values[key]) ?? fallback;
    const kind = (key: string, fallback: FeeKind): FeeKind =>
      values[key] === 'percent' || values[key] === 'amount' ? (values[key] as FeeKind) : fallback;
    input = {
      principal: number('principal', input.principal),
      currentRate: number('currentRate', input.currentRate),
      newRate: number('newRate', input.newRate),
      days: number('days', input.days),
      entryFee: { kind: kind('entryFeeKind', input.entryFee.kind), value: number('entryFee', input.entryFee.value) },
      exitFee: { kind: kind('exitFeeKind', input.exitFee.kind), value: number('exitFee', input.exitFee.value) },
    };
    marks.take(result);
  }

  const fees = [
    { key: 'entryFee', label: 'Entry fee', hint: 'Paid converting into the new coin or place.' },
    { key: 'exitFee', label: 'Exit fee', hint: 'Paid converting back at the end.' },
  ] as const;
  const kinds: { kind: FeeKind; label: string }[] = [
    { kind: 'percent', label: '%' },
    { kind: 'amount', label: UNIT },
  ];
</script>

<PageHeader title="Stablecoin Switch" subtitle="Is a higher rate elsewhere worth the conversion fees?">
  {#snippet actions()}
    <Button variant="ghost" icon="refresh" onclick={reset}>Reset to example</Button>
  {/snippet}
</PageHeader>

<div class="grid grid-cols-1 gap-4 lg:grid-cols-5">
  <Card title="Your figures" subtitle="Saved in this browser only" class="lg:col-span-2">
    <FillBox form="stablecoin" class="mb-3" current={flat} onfill={fill} />
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <div class="sm:col-span-2">
        <label class="label" for="switch-principal"
          >Amount ({UNIT})<FilledMark show={marks.has('principal', input.principal)} /></label
        >
        <input
          id="switch-principal"
          class="field tabular-nums"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          aria-invalid={!!errors.principal}
          aria-describedby={errors.principal ? 'switch-principal-error' : undefined}
          bind:value={() => input.principal, (value) => (input.principal = numberOf(value))}
        />
        {#if errors.principal}<p id="switch-principal-error" class="mt-1 text-xs text-rose-700 dark:text-rose-400">
            {errors.principal}
          </p>{/if}
      </div>
      <div>
        <label class="label" for="switch-current-rate"
          >Current rate (% a year)<FilledMark show={marks.has('currentRate', input.currentRate)} /></label
        >
        <input
          id="switch-current-rate"
          class="field tabular-nums"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          aria-invalid={!!errors.currentRate}
          aria-describedby={errors.currentRate ? 'switch-current-rate-error' : undefined}
          bind:value={() => input.currentRate, (value) => (input.currentRate = numberOf(value))}
        />
        {#if errors.currentRate}<p id="switch-current-rate-error" class="mt-1 text-xs text-rose-700 dark:text-rose-400">
            {errors.currentRate}
          </p>{/if}
      </div>
      <div>
        <label class="label" for="switch-new-rate"
          >New rate (% a year)<FilledMark show={marks.has('newRate', input.newRate)} /></label
        >
        <input
          id="switch-new-rate"
          class="field tabular-nums"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          aria-invalid={!!errors.newRate}
          aria-describedby={errors.newRate ? 'switch-new-rate-error' : undefined}
          bind:value={() => input.newRate, (value) => (input.newRate = numberOf(value))}
        />
        {#if errors.newRate}<p id="switch-new-rate-error" class="mt-1 text-xs text-rose-700 dark:text-rose-400">
            {errors.newRate}
          </p>{/if}
      </div>
      <div class="sm:col-span-2">
        <label class="label" for="switch-days">Days held<FilledMark show={marks.has('days', input.days)} /></label>
        <input
          id="switch-days"
          class="field tabular-nums"
          type="number"
          inputmode="numeric"
          min="1"
          step="1"
          aria-invalid={!!errors.days}
          aria-describedby={errors.days ? 'switch-days-error' : undefined}
          bind:value={() => input.days, (value) => (input.days = numberOf(value))}
        />
        {#if errors.days}<p id="switch-days-error" class="mt-1 text-xs text-rose-700 dark:text-rose-400">
            {errors.days}
          </p>{/if}
      </div>
      {#each fees as fee (fee.key)}
        <div>
          <label class="label" for="switch-{fee.key}"
            >{fee.label}<FilledMark show={marks.has(fee.key, input[fee.key].value)} /></label
          >
          <div class="flex gap-1.5">
            <input
              id="switch-{fee.key}"
              class="field min-w-0 tabular-nums"
              type="number"
              inputmode="decimal"
              min="0"
              step="any"
              aria-invalid={!!errors[fee.key]}
              aria-describedby="switch-{fee.key}-hint"
              bind:value={() => input[fee.key].value, (value) => (input[fee.key].value = numberOf(value))}
            />
            <select
              class="field w-auto shrink-0"
              aria-label="{fee.label} as"
              bind:value={() => input[fee.key].kind, (kind) => (input[fee.key].kind = kind)}
            >
              {#each kinds as option (option.kind)}<option value={option.kind}>{option.label}</option>{/each}
            </select>
          </div>
          <p
            id="switch-{fee.key}-hint"
            class="mt-1 text-xs {errors[fee.key] ? 'text-rose-700 dark:text-rose-400' : 'muted'}"
          >
            {errors[fee.key] ?? fee.hint}
          </p>
        </div>
      {/each}
    </div>
  </Card>

  <div class="min-w-0 space-y-4 lg:col-span-3">
    {#if result.ok}
      <section class="panel p-5" aria-labelledby="switch-verdict" aria-live="polite" data-decision={result.decision}>
        <p class="muted text-[0.7rem] font-semibold tracking-[0.12em] uppercase">Decision</p>
        <h2 id="switch-verdict" class="mt-1 font-serif text-2xl font-semibold tracking-tight">
          {result.decision === 'switch' ? 'Switch' : 'Stay'}
        </h2>
        <p class="mt-1 text-sm">
          {#if result.decision === 'switch'}
            Over {input.days} days the higher rate earns more than the fees cost: switching ends
            <Money value={result.net} currency={UNIT} tone /> ahead.
          {:else}
            Over {input.days} days the fees cost more than the higher rate earns: switching ends
            <Money value={result.net} currency={UNIT} tone /> behind.
          {/if}
        </p>
        <p class="muted mt-2 text-sm">
          {#if result.breakEvenDays === null}
            The new rate is not far enough above the current one to ever earn the fees back.
          {:else if result.breakEvenDays <= 1}
            Switching is ahead from the first day.
          {:else}
            Switching pays off from day {result.breakEvenDays}: hold at least {result.breakEvenDays} days.
          {/if}
        </p>
      </section>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Stat label="Value if you stay">
          <Money value={result.stay.value} currency={UNIT} />
        </Stat>
        <Stat label="Value if you switch">
          <Money value={result.switch.value} currency={UNIT} />
        </Stat>
        <Stat label="Net gain or loss">
          <Money value={result.net} currency={UNIT} signed tone />
        </Stat>
      </div>

      <Card title="Where the difference comes from">
        <dl class="grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 text-sm tabular-nums">
          <dt>Interest if you stay</dt>
          <dd class="text-right"><Money value={result.stay.interest} currency={UNIT} /></dd>
          <dt>Interest if you switch</dt>
          <dd class="text-right"><Money value={result.switch.interest} currency={UNIT} /></dd>
          <dt>Entry fee</dt>
          <dd class="text-right"><Money value={-result.switch.entryFee} currency={UNIT} /></dd>
          <dt>Exit fee</dt>
          <dd class="text-right"><Money value={-result.switch.exitFee} currency={UNIT} /></dd>
        </dl>
        <p class="muted mt-4 text-xs">
          Interest is simple: rate × days / 365. Rates on stablecoin products move often and are not guaranteed, so
          check both before you move.
        </p>
      </Card>
    {:else}
      <section class="panel p-5" aria-live="polite">
        <h2 class="font-serif text-lg font-semibold">Check your figures</h2>
        <p class="muted mt-1 text-sm">Fix the highlighted fields to see the comparison.</p>
      </section>
    {/if}
  </div>
</div>
