<script lang="ts">
  let { title, detail }: { title: string; detail?: string } = $props();
</script>

<div class="flex flex-col items-center justify-center px-4 py-10 text-center">
  <p class="font-medium text-stone-700 dark:text-stone-200">{title}</p>
  {#if detail}<p class="muted mt-1 max-w-md text-sm">{detail}</p>{/if}
</div>
