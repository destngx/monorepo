<script lang="ts">
  import type { Project } from '$lib/content/schema';

  type Props = { project: Project; isLarge?: boolean };

  let { project, isLarge = false }: Props = $props();

  const statusStyles: Record<Project['status'], string> = {
    active: 'bg-teal text-paper',
    shipped: 'bg-ink text-paper',
    experiment: 'bg-accent text-accent-ink',
    archived: 'bg-paper text-ink-soft',
  };
</script>

<article class="panel panel-hover group flex h-full flex-col overflow-hidden">
  {#if project.sfx}
    <span
      class={[
        'sfx absolute -top-2 right-3 rotate-[8deg] text-5xl transition-transform duration-300',
        'group-hover:scale-125 group-hover:rotate-[-4deg]',
        isLarge && 'md:text-7xl',
      ]}
      aria-hidden="true">{project.sfx}</span
    >
  {/if}

  <div class={['flex flex-1 flex-col gap-4 p-6', isLarge && 'md:p-8']}>
    <div class="flex flex-wrap items-center gap-2 font-mono text-xs">
      <span class="chip {statusStyles[project.status]}">{project.status}</span>
      <span class="text-ink-soft">{project.year}</span>
    </div>

    <h3 class={['pr-16 font-display text-2xl leading-tight', isLarge && 'md:text-4xl']}>
      <a href="/projects/{project.slug}" class="after:absolute after:inset-0" data-cursor="read">{project.title}</a>
    </h3>

    <p class={['text-ink-soft', isLarge && 'md:text-lg']}>{project.summary}</p>

    <ul class="mt-auto flex flex-wrap gap-1.5 pt-2" aria-label="Tech stack">
      {#each project.stack as tech (tech)}
        <li class="chip">{tech}</li>
      {/each}
    </ul>
  </div>
</article>
