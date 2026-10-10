<script lang="ts">
  import { reveal } from '$lib/motion/reveal';
  import ProjectCard from '$lib/ui/ProjectCard.svelte';
  import Seo from '$lib/ui/Seo.svelte';

  let { data } = $props();
</script>

<Seo
  title="Work"
  description="Side projects and tools I've built: AI infrastructure, finance tooling, and native macOS apps."
/>

<header class="pt-16 pb-12">
  <p class="chapter-label">Index</p>
  <h1 class="mt-2 font-display text-5xl md:text-7xl">The work</h1>
  <p class="mt-6 max-w-2xl text-lg text-ink-soft">
    Everything here lives in one public <a
      class="ink-link"
      href="https://github.com/destngx/monorepo"
      target="_blank"
      rel="noopener">monorepo</a
    >. Each one started as an itch I wanted to scratch.
  </p>
</header>

<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
  {#each data.projects as project, index (project.slug)}
    <div use:reveal={{ delay: (index % 3) * 80 }}>
      <ProjectCard {project} />
    </div>
  {/each}
</div>
