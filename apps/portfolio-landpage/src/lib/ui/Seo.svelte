<script lang="ts">
  import { page } from '$app/state';
  import { site } from '$lib/data/site';

  type Props = {
    title?: string;
    description?: string;
    type?: 'website' | 'article';
    image?: string;
    publishedAt?: Date;
  };

  let { title, description = site.description, type = 'website', image = '/og.png', publishedAt }: Props = $props();

  const fullTitle = $derived(title ? `${title} | ${site.name}` : site.title);
  const canonical = $derived(new URL(page.url.pathname, site.url).href);
  const imageUrl = $derived(new URL(image, site.url).href);
</script>

<svelte:head>
  <title>{fullTitle}</title>
  <meta name="description" content={description} />
  <link rel="canonical" href={canonical} />
  <meta property="og:type" content={type} />
  <meta property="og:site_name" content={site.name} />
  <meta property="og:title" content={fullTitle} />
  <meta property="og:description" content={description} />
  <meta property="og:url" content={canonical} />
  <meta property="og:image" content={imageUrl} />
  <meta name="twitter:card" content="summary_large_image" />
  {#if publishedAt}
    <meta property="article:published_time" content={publishedAt.toISOString()} />
  {/if}
</svelte:head>
