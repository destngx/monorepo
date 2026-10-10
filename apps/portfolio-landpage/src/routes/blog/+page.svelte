<script lang="ts">
  import PostRow from '$lib/ui/PostRow.svelte';
  import Seo from '$lib/ui/Seo.svelte';

  let { data } = $props();
</script>

<Seo title="Writing" description="Notes on DevOps, infrastructure, and the side projects I build." />

<header class="pt-16 pb-10">
  <p class="chapter-label">Notes</p>
  <h1 class="mt-2 font-display text-5xl md:text-7xl">Writing</h1>
  <p class="mt-6 max-w-2xl text-lg text-ink-soft">
    Notes from running infrastructure and building side projects. Also available as <a class="ink-link" href="/rss.xml"
      >RSS</a
    >.
  </p>
</header>

{#if data.posts.length}
  <div class="border-t-[3px] border-dashed border-rule">
    {#each data.posts as post (post.slug)}
      <PostRow {post} />
    {/each}
  </div>
{:else}
  <p class="bubble inline-block text-lg">First issue is still at the printer. Check back soon.</p>
{/if}
