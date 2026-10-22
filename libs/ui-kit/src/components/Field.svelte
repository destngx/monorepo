<script lang="ts">
  import type { Snippet } from 'svelte';

  // A labelled form control; the control itself is the child, with the given id.
  let { id, label, hint, children }: { id: string; label: string; hint?: string; children: Snippet } = $props();
</script>

<div>
  <label class="label" for={id}>{label}</label>
  {@render children()}
  {#if hint}<p class="muted mt-1 text-xs">{hint}</p>{/if}
</div>
