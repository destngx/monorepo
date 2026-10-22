<script lang="ts">
  import Icon from './Icon.svelte';

  // The chevron that folds or opens one page of a page list.
  let { open, label, controls, ontoggle }: { open: boolean; label: string; controls: string; ontoggle: () => void } =
    $props();
</script>

<button
  type="button"
  class="icon-btn -ml-1"
  aria-expanded={open}
  aria-controls={controls}
  aria-label="{open ? 'Fold' : 'Open'} {label}"
  title="{open ? 'Fold' : 'Open'} {label}"
  onclick={ontoggle}
>
  <Icon name="chevronDown" size={16} class="transition-transform {open ? '' : '-rotate-90'}" />
</button>
