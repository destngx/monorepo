<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  let { title, subtitle, children }: { title: string; subtitle?: string; children: Snippet } = $props();
</script>

<section class="panel-strong w-full max-w-md p-6 sm:p-8">
  <div class="mb-6 flex items-center gap-3">
    <span class="grid size-10 place-items-center rounded-xl bg-accent-600 text-ink-50"
      ><Icon name="activity" size={22} /></span
    >
    <div>
      <h1 class="text-lg font-semibold">{title}</h1>
      {#if subtitle}<p class="muted text-sm">{subtitle}</p>{/if}
    </div>
  </div>
  {@render children()}
</section>
