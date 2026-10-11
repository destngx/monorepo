<script lang="ts">
  import '../app.css';
  import { dev } from '$app/environment';
  import { onNavigate } from '$app/navigation';
  import { injectAnalytics } from '@vercel/analytics/sveltekit';
  import Aurora from '$lib/ui/Aurora.svelte';
  import InkCursor from '$lib/ui/InkCursor.svelte';
  import IntroSplash from '$lib/ui/IntroSplash.svelte';
  import SiteFooter from '$lib/ui/SiteFooter.svelte';
  import SiteHeader from '$lib/ui/SiteHeader.svelte';
  import SoundCloudPlayer from '$lib/ui/SoundCloudPlayer.svelte';

  let { children } = $props();

  injectAnalytics({ mode: dev ? 'development' : 'production' });

  // Page-turn transition between routes, where the View Transitions API is supported.
  onNavigate((navigation) => {
    if (!document.startViewTransition || navigation.from?.url.pathname === navigation.to?.url.pathname) return;

    return new Promise((resolve) => {
      document.startViewTransition(async () => {
        resolve();
        await navigation.complete;
      });
    });
  });
</script>

<a href="#main" class="btn btn-primary fixed top-4 left-4 z-[110] -translate-y-24 focus:translate-y-0"
  >Skip to content</a
>

<Aurora />
<IntroSplash />
<InkCursor />
<SiteHeader />

<!-- Panels intentionally bleed past the content column; clip (not hide) so nothing becomes a scroll container. -->
<main id="main" class="overflow-x-clip">
  <div class="mx-auto max-w-6xl px-4 sm:px-6">
    {@render children()}
  </div>
</main>

<SiteFooter />
<SoundCloudPlayer />
