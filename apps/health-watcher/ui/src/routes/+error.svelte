<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
</script>

<div class="mx-auto max-w-lg px-4 py-16 text-center">
  <p class="text-5xl font-semibold text-teal-600 dark:text-teal-400">{page.status}</p>
  <p class="mt-3 text-slate-700 dark:text-slate-200">{page.error?.message ?? 'Something went wrong.'}</p>
  <a class="btn-primary mt-6" href="{base}/">Back to home</a>
</div>
