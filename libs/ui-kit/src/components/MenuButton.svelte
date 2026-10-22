<script lang="ts" module>
  import type { IconName } from './Icon.svelte';

  export interface MenuItem {
    label: string;
    icon?: IconName;
    /** Extra classes, e.g. to show an item only on touch screens. */
    class?: string;
    onselect: () => void;
  }
</script>

<script lang="ts">
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  // A button that opens a short list of actions below it. Escape or a click
  // outside closes it; arrow keys move between the items.
  interface Props {
    label: string;
    icon?: IconName;
    items: MenuItem[];
    busy?: boolean;
    busyText?: string;
  }

  let { label, icon, items, busy = false, busyText }: Props = $props();

  let open = $state(false);
  let root: HTMLDivElement | undefined = $state();
  let list: HTMLUListElement | undefined = $state();

  function buttons(): HTMLButtonElement[] {
    // Items hidden for this device have no layout box.
    return [...(list?.querySelectorAll('button') ?? [])].filter((b) => b.offsetParent !== null);
  }

  $effect(() => {
    if (!open) return;
    buttons()[0]?.focus();
    const outside = (e: PointerEvent) => {
      if (!root?.contains(e.target as Node)) open = false;
    };
    document.addEventListener('pointerdown', outside);
    return () => document.removeEventListener('pointerdown', outside);
  });

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      open = false;
      root?.querySelector<HTMLButtonElement>('button')?.focus();
      return;
    }
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    e.preventDefault();
    const all = buttons();
    const at = all.indexOf(document.activeElement as HTMLButtonElement);
    all[(at + (e.key === 'ArrowDown' ? 1 : all.length - 1)) % all.length]?.focus();
  }

  function select(item: MenuItem) {
    open = false;
    item.onselect();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="relative inline-flex" bind:this={root} onkeydown={keydown}>
  <Button {icon} {busy} {busyText} aria-haspopup="menu" aria-expanded={open} onclick={() => (open = !open)}>
    {label}<Icon name="chevronDown" size={14} />
  </Button>
  {#if open}
    <ul
      bind:this={list}
      role="menu"
      class="panel-strong absolute top-full right-0 z-20 mt-1 min-w-48 py-1 text-sm shadow-brut"
    >
      {#each items as item (item.label)}
        <li role="none" class={item.class ?? 'flex'}>
          <button
            type="button"
            role="menuitem"
            class="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-ink-900/5 focus-visible:bg-ink-900/5 focus-visible:outline-none dark:hover:bg-ink-50/10 dark:focus-visible:bg-ink-50/10"
            onclick={() => select(item)}
          >
            {#if item.icon}<Icon name={item.icon} size={16} />{/if}{item.label}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
