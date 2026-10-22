<script lang="ts">
  import type { CashFlow, CategoryNature, ExpenseSplit } from '$lib/api/types';
  import { ESSENTIAL, FIXED, FLEXIBLE, UNTAGGED } from '$lib/charts/palette';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { expenseSplitMessage } from './yearly';

  interface Props {
    split: ExpenseSplit;
    /** Completed months, to tell "too few months" from "nothing to save". */
    months: CashFlow[];
  }

  // WM-DASH-8: needs (fixed + essential) vs wants (flexible), and the room to save.
  let { split, months }: Props = $props();

  const ROOM_SHOWN = 5;
  const UNTAGGED_SHOWN = 8;
  const TYPICAL_MIN_MONTHS = 4;
  const natureLabel: Record<CategoryNature, string> = {
    fixed: 'Fixed',
    essential: 'Essential',
    flexible: 'Flexible',
    lending: 'Lending',
  };

  const message = $derived(expenseSplitMessage(split));
  const style = $derived(STATUS[message.tone]);
  const spending = $derived(split.fixed.total + split.essential.total + split.flexible.total + split.untagged.total);
  const tagged = $derived(split.fixed.total + split.essential.total + split.flexible.total > 0);
  const groups = $derived(
    [
      { name: 'Fixed', about: 'set by contracts', color: FIXED, total: split.fixed.total },
      { name: 'Essential', about: 'needed, can be lowered', color: ESSENTIAL, total: split.essential.total },
      { name: 'Flexible', about: 'wants', color: FLEXIBLE, total: split.flexible.total },
      { name: 'Not tagged', about: '', color: UNTAGGED, total: split.untagged.total },
    ].filter((group) => group.total > 0),
  );
  const share = (value: number) => (spending > 0 ? (value / spending) * 100 : 0);
  const activeMonths = $derived(months.filter((m) => m.expense > 0).length);
</script>

<Card title="Needs vs wants" subtitle="Where your money must go, and where it can be saved" class="mt-4">
  <div class="border-l-4 pl-3 {style.edge}">
    <p class="flex items-center gap-2 font-semibold {style.ink}"><Icon name={style.icon} />{message.title}</p>
    <p class="mt-1 text-sm">{message.text}</p>
    {#if message.detail}<p class="muted mt-1 text-sm">{message.detail}</p>{/if}
  </div>

  {#if tagged}
    <div class="mt-4 flex h-3 gap-0.5 overflow-hidden rounded" role="img" aria-label="Spending split by nature">
      {#each groups as group (group.name)}
        <div
          class="h-full first:rounded-l last:rounded-r"
          style:width="{share(group.total)}%"
          style:background={group.color}
        ></div>
      {/each}
    </div>
    <ul class="mt-3 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4" data-testid="split-legend">
      {#each groups as group (group.name)}
        <li class="flex items-start gap-2">
          <span class="mt-1 size-2.5 shrink-0 rounded-sm" style:background={group.color}></span>
          <span>
            <span class="font-medium">{group.name}</span>
            {#if group.about}<span class="muted text-xs">({group.about})</span>{/if}
            <span class="block"><Money value={group.total} /> · {formatPercent(share(group.total))}</span>
          </span>
        </li>
      {/each}
    </ul>

    <h3 class="mt-5 text-sm font-semibold">Room to save</h3>
    {#if split.room.length}
      <p class="mt-1 text-sm" data-testid="room-total">
        Living like your leaner months would free about <Money value={split.roomPerMonth} /> a month, <Money
          value={split.roomPerYear}
        /> a year.
      </p>
      <p class="muted mt-1 text-xs">
        Leaner months are whole months you have already lived, so essentials are lowered, not cut. Below is where it
        comes from; categories rarely dip in the same month, so the lines add up to more than the total.
      </p>
      <ul class="mt-2 divide-y divide-stone-900/5 text-sm dark:divide-white/5" data-testid="room">
        {#each split.room.slice(0, ROOM_SHOWN) as room (room.category)}
          <li class="flex items-baseline justify-between gap-3 py-2">
            <span>
              <span class="font-medium">{room.category}</span>
              <span class="muted text-xs">{natureLabel[room.nature]}</span>
              <span class="muted block text-xs"
                >typically <Money value={room.typical} />, <Money value={room.lean} /> in leaner months</span
              >
            </span>
            <span class="tabular whitespace-nowrap"><Money value={room.perMonth} /> / month</span>
          </li>
        {/each}
      </ul>
    {:else if activeMonths < TYPICAL_MIN_MONTHS}
      <p class="muted mt-1 text-sm">
        Needs {TYPICAL_MIN_MONTHS} completed months with spending to find your leaner months.
      </p>
    {:else}
      <p class="muted mt-1 text-sm">Your essential and flexible spending is steady month to month.</p>
    {/if}
  {/if}

  {#if split.untaggedCategories.length}
    <p class="muted mt-4 text-sm" data-testid="untagged">
      Not tagged yet: {split.untaggedCategories.slice(0, UNTAGGED_SHOWN).join(', ')}{split.untaggedCategories.length >
      UNTAGGED_SHOWN
        ? ` and ${split.untaggedCategories.length - UNTAGGED_SHOWN} more`
        : ''}. Write fixed, essential or flexible next to them in column B of the Categories tab.
    </p>
  {/if}
</Card>
