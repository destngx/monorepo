<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import { action } from '$lib/state/action.svelte';
  import Icon, { type IconName } from './Icon.svelte';
  import Spinner from './Spinner.svelte';

  // The shared button (see docs/wealth-management/AGENTS.md "Loading and feedback"):
  // variants map to the .btn-* styles. An onclick that returns a promise makes
  // the button busy until it settles (spinner, aria-busy, further clicks
  // ignored), then toasts `success` or the error. `busy` forces the state for
  // work started elsewhere. Icon-only buttons need `label` (aria-label + tooltip).
  interface Props extends Omit<HTMLButtonAttributes, 'onclick' | 'children'> {
    variant?: 'primary' | 'ghost' | 'danger';
    size?: 'md' | 'sm' | 'icon';
    icon?: IconName;
    /** Accessible name and tooltip; required when there is no visible text. */
    label?: string;
    busy?: boolean;
    /** Text shown while busy, e.g. "Saving…" (defaults to the normal text). */
    busyText?: string;
    onclick?: (event: MouseEvent) => unknown;
    success?: string;
    errorContext?: string;
    children?: Snippet;
  }

  let {
    variant = 'ghost',
    size = 'md',
    icon,
    label,
    busy = false,
    busyText,
    onclick,
    success,
    errorContext,
    children,
    disabled,
    type = 'button',
    class: className = '',
    ...rest
  }: Props = $props();

  const run = action((pending: Promise<unknown>) => pending, {
    get success() {
      return success;
    },
    get errorContext() {
      return errorContext;
    },
  });
  const isBusy = $derived(busy || run.busy);
  const variants = { primary: 'btn-primary', ghost: 'btn-ghost', danger: 'btn-danger' };
  const sizes = { md: '', sm: 'px-2.5 py-1 text-xs', icon: 'px-2' };

  function handle(event: MouseEvent) {
    if (isBusy) {
      event.preventDefault();
      return;
    }
    const result = onclick?.(event);
    // Only async handlers get the busy state and toasts; sync ones run as they are.
    if (result instanceof Promise) void run.run(result);
  }
</script>

<button
  {...rest}
  {type}
  {disabled}
  class="{variants[variant]} {sizes[size]} {className}"
  aria-label={label}
  title={rest.title ?? label}
  aria-busy={isBusy || undefined}
  aria-disabled={isBusy || undefined}
  data-busy={isBusy || undefined}
  onclick={handle}
>
  {#if isBusy}
    <Spinner size={size === 'sm' ? 14 : 16} />
  {:else if icon}
    <Icon name={icon} size={size === 'sm' ? 14 : 16} />
  {/if}
  {#if isBusy && busyText}{busyText}{:else if children}{@render children()}{/if}
</button>
