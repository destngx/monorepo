<script lang="ts">
  import Seo from '$lib/ui/Seo.svelte';

  let { data } = $props();

  const meta = $derived(data.meta);
  const Content = $derived(data.content);
</script>

<Seo title={meta.title} description={meta.summary} type="article" />

<article class="mx-auto max-w-3xl pt-12">
  <a href="/projects" class="font-mono text-sm font-semibold uppercase tracking-wider ink-link">← All work</a>

  <header class="panel halftone relative mt-8 overflow-hidden p-8 md:p-10">
    <div class="speedlines absolute inset-0 opacity-10" aria-hidden="true"></div>
    {#if meta.sfx}
      <span class="sfx absolute -top-1 right-4 rotate-12 text-6xl md:text-8xl" aria-hidden="true">{meta.sfx}</span>
    {/if}
    <p class="chapter-label relative">Case file / {meta.year} / {meta.status}</p>
    <h1 class="relative mt-3 pr-20 font-display text-4xl leading-tight md:text-6xl">{meta.title}</h1>
    <p class="relative mt-5 text-lg text-ink-soft md:text-xl">{meta.summary}</p>

    <ul class="relative mt-6 flex flex-wrap gap-1.5" aria-label="Tech stack">
      {#each meta.stack as tech (tech)}
        <li class="chip">{tech}</li>
      {/each}
    </ul>

    {#if meta.repo || meta.live}
      <div class="relative mt-8 flex flex-wrap gap-3">
        {#if meta.live}<a class="btn btn-primary" href={meta.live} target="_blank" rel="noopener">Visit live ↗</a>{/if}
        {#if meta.repo}<a class="btn btn-ghost" href={meta.repo} target="_blank" rel="noopener">Source ↗</a>{/if}
      </div>
    {/if}
  </header>

  <div class="prose-manga mt-12">
    <Content />
  </div>
</article>
