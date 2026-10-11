<script lang="ts">
  import { formatDate } from '$lib/format';
  import Seo from '$lib/ui/Seo.svelte';

  let { data } = $props();

  const post = $derived(data.post);
  const Content = $derived(data.content);
</script>

<Seo title={post.title} description={post.description} type="article" publishedAt={post.date} />

<article class="mx-auto max-w-3xl pt-12">
  <a href="/blog" class="font-mono text-sm font-semibold uppercase tracking-wider ink-link">← All writing</a>

  <header class="mt-8 border-b-[3px] border-rule pb-8">
    <p class="chapter-label">
      <time datetime={post.date.toISOString()}>{formatDate(post.date)}</time>
      {#if post.tags.length}/ {post.tags.join(', ')}{/if}
      {#if post.draft}<span class="chip ml-2">draft</span>{/if}
    </p>
    <h1 class="mt-3 font-display text-4xl leading-tight md:text-6xl">{post.title}</h1>
    <p class="mt-5 text-lg text-ink-soft md:text-xl">{post.description}</p>
  </header>

  <div class="prose-manga mt-10">
    <Content />
  </div>
</article>
