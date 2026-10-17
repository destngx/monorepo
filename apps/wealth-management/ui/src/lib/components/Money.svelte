<script lang="ts">
  import { formatMoney, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { MASK_TOKEN_CLASS, maskToggle } from './maskTokens';

  interface Props {
    value: number | null | undefined;
    currency?: string;
    /** Prefix + for positive values. */
    signed?: boolean;
    /** Colour by sign: green for positive, orange for negative. */
    tone?: boolean;
    class?: string;
  }

  let { value, currency = 'VND', signed = false, tone = false, class: className = '' }: Props = $props();

  const text = $derived.by(() => {
    const formatted = formatMoney(value, currency);
    return signed && value !== null && value !== undefined && value > 0 ? `+${formatted}` : formatted;
  });
  const toneClass = $derived(
    !tone || !value
      ? ''
      : value > 0
        ? 'text-emerald-600 dark:text-emerald-400'
        : 'text-orange-600 dark:text-orange-400',
  );
</script>

<!-- Any amount toggles the global privacy mask (WM-UI-2). -->
<button
  type="button"
  class="{MASK_TOKEN_CLASS} {toneClass} {className}"
  title={mask.revealed ? 'Hide amounts' : 'Show amounts'}
  aria-label={mask.revealed ? text : 'Hidden amount, click to reveal'}
  {@attach maskToggle}>{mask.revealed ? text : MASK}</button
>
