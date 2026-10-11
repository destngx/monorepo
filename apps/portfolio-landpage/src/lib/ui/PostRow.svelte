<script lang="ts">
  import type { Post } from '$lib/content/schema';
  import { formatDate } from '$lib/format';

  type Props = { post: Post };

  let { post }: Props = $props();
</script>

<article
  class="group relative grid gap-1 border-b-[3px] border-dashed border-rule py-5 sm:grid-cols-[8rem_1fr] sm:gap-6"
>
  <time class="font-mono text-sm text-ink-soft" datetime={post.date.toISOString()}>{formatDate(post.date)}</time>
  <div>
    <h3 class="font-display text-xl transition-colors group-hover:text-accent">
      <a href="/blog/{post.slug}" class="after:absolute after:inset-0" data-cursor="read">{post.title}</a>
      {#if post.draft}<span class="chip ml-2 align-middle">draft</span>{/if}
    </h3>
    <p class="mt-1 text-ink-soft">{post.description}</p>
  </div>
</article>
