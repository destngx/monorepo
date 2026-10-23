<script lang="ts">
  import { Icon } from '@libs/ui-kit';
  import { currentMonth, formatMonth } from '$lib/format';

  // A month picker that works in every browser (Safari and Firefox on the
  // desktop have no `type="month"` picker): a button that opens a year with
  // its twelve months. The value is "yyyy-mm", or '' for none. The panel
  // opens in the flow, not floating, so a modal's scrolling body never
  // clips it.
  interface Props {
    id: string;
    value: string;
    /** Text on the button when no month is picked. */
    placeholder?: string;
    /** Label of the button that clears the month. */
    clearLabel?: string;
    describedby?: string;
  }

  let { id, value = $bindable(), placeholder = 'Pick a month', clearLabel = 'Clear', describedby }: Props = $props();

  const MONTHS = Array.from({ length: 12 }, (_, i) => String(i + 1).padStart(2, '0'));
  const today = currentMonth();

  let open = $state(false);
  let year = $state(Number(today.slice(0, 4)));
  let root: HTMLDivElement | undefined = $state();
  let panel: HTMLDivElement | undefined = $state();

  function toggle() {
    if (!open) year = Number((value || today).slice(0, 4));
    open = !open;
  }

  function pick(month: string) {
    value = `${year}-${month}`;
    close();
  }

  function clear() {
    value = '';
    close();
  }

  function close() {
    open = false;
    root?.querySelector<HTMLButtonElement>(`#${id}`)?.focus();
  }

  $effect(() => {
    if (!open) return;
    (panel?.querySelector<HTMLButtonElement>('[aria-pressed="true"]') ?? panel?.querySelector('button'))?.focus();
    const outside = (e: PointerEvent) => {
      if (!root?.contains(e.target as Node)) open = false;
    };
    document.addEventListener('pointerdown', outside);
    return () => document.removeEventListener('pointerdown', outside);
  });

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      // Close the picker, not the modal around it.
      e.preventDefault();
      e.stopPropagation();
      close();
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div bind:this={root} onkeydown={keydown}>
  <button
    {id}
    type="button"
    class="field flex items-center justify-between gap-2 text-left"
    aria-haspopup="dialog"
    aria-expanded={open}
    aria-describedby={describedby}
    onclick={toggle}
  >
    <span class={value ? '' : 'muted'}>{value ? formatMonth(value) : placeholder}</span>
    <Icon name="calendar" size={16} />
  </button>
  {#if open}
    <div bind:this={panel} role="dialog" aria-label="Choose a month" class="panel-strong mt-1 w-full max-w-64 p-3">
      <div class="mb-2 flex items-center justify-between">
        <button type="button" class="icon-btn" aria-label="Previous year" onclick={() => year--}
          ><Icon name="chevronLeft" size={16} /></button
        >
        <span class="font-semibold tabular-nums" aria-live="polite">{year}</span>
        <button type="button" class="icon-btn" aria-label="Next year" onclick={() => year++}
          ><Icon name="chevronRight" size={16} /></button
        >
      </div>
      <div class="grid grid-cols-3 gap-1" role="group" aria-label="Months of {year}">
        {#each MONTHS as month (month)}
          {@const key = `${year}-${month}`}
          <button
            type="button"
            aria-pressed={key === value}
            aria-label={formatMonth(key)}
            class="rounded-md px-2 py-1.5 text-sm {key === value
              ? 'bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900'
              : key === today
                ? 'ring-1 ring-accent-400 hover:bg-stone-900/5 dark:hover:bg-white/10'
                : 'hover:bg-stone-900/5 dark:hover:bg-white/10'}"
            onclick={() => pick(month)}>{formatMonth(key, true)}</button
          >
        {/each}
      </div>
      <div class="mt-2 flex justify-between gap-2 text-xs">
        <button type="button" class="link" onclick={() => ((year = Number(today.slice(0, 4))), pick(today.slice(5)))}
          >This month</button
        >
        {#if value}<button type="button" class="link" onclick={clear}>{clearLabel}</button>{/if}
      </div>
    </div>
  {/if}
</div>
