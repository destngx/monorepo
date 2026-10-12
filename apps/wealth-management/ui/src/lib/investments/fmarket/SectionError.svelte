<script lang="ts">
  let { title, message }: { title: string; message: string } = $props();
</script>

<div class="rounded-xl border border-amber-500/40 p-3 text-sm text-amber-800 dark:text-amber-200" role="alert">
  <p class="font-medium">{title}</p>
  <p class="mt-0.5 text-xs break-words opacity-80">{message}</p>
</div>
