<script lang="ts">
  import { page } from '$app/state';
  import Seo from '$lib/ui/Seo.svelte';
</script>

<Seo title={String(page.status)} description="Something went sideways." />

<section class="relative grid min-h-[60vh] place-items-center py-20 text-center">
  <div class="speedlines absolute inset-0 -z-10 opacity-20" aria-hidden="true"></div>
  <div>
    <p class="sfx text-7xl md:text-9xl" aria-hidden="true">ガーン</p>
    <h1 class="mt-6 font-display text-6xl md:text-8xl">{page.status}</h1>
    <p class="bubble mx-auto mt-8 max-w-md text-lg">
      {page.status === 404 ? "This page wandered off to find Korok seeds. It isn't here." : page.error?.message}
    </p>
    <a class="btn btn-primary mt-12" href="/">Back to page one</a>
  </div>
</section>
