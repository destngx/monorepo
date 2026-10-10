<!--
  First-visit title card: DEST / NGUYXN stamps in over speed lines, then the panel
  wipes away. Client-only and once per session, so it never hides content from
  crawlers or no-JS visitors. Skipped entirely for reduced motion.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { prefersReducedMotion } from 'svelte/motion';

  const SESSION_KEY = 'intro-seen';
  const DURATION_MS = 1900;

  let phase = $state<'hidden' | 'playing' | 'leaving'>('hidden');

  const finish = () => {
    if (phase !== 'playing') return;
    phase = 'leaving';
    setTimeout(() => (phase = 'hidden'), 450);
  };

  onMount(() => {
    let hasSeen = false;
    try {
      hasSeen = sessionStorage.getItem(SESSION_KEY) === '1';
      sessionStorage.setItem(SESSION_KEY, '1');
    } catch (error) {
      console.warn('sessionStorage unavailable, intro will play every visit', error);
    }
    if (hasSeen || prefersReducedMotion.current) return;

    phase = 'playing';
    const timer = setTimeout(finish, DURATION_MS);
    return () => clearTimeout(timer);
  });
</script>

<svelte:window onkeydown={(event) => event.key === 'Escape' && finish()} />

{#if phase !== 'hidden'}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="intro fixed inset-0 z-[100] grid place-items-center bg-paper"
    class:leaving={phase === 'leaving'}
    aria-hidden="true"
    onclick={finish}
  >
    <div class="speedlines absolute inset-0 opacity-25"></div>
    <div class="relative text-center">
      <p class="stamp stamp-1 font-display text-[22vw] leading-[0.85] md:text-[14vw]">DEST</p>
      <p class="stamp stamp-2 font-display text-[22vw] leading-[0.85] text-accent md:text-[14vw]">NGUYXN</p>
      <p class="sfx stamp-3 absolute -top-8 -right-4 rotate-12 text-6xl md:text-8xl">ドン!</p>
    </div>
    <p class="absolute bottom-8 font-mono text-xs tracking-[0.3em] text-ink-soft uppercase">click to skip</p>
  </div>
{/if}

<style>
  .intro {
    clip-path: inset(0 0 0 0);
    transition: clip-path 450ms cubic-bezier(0.7, 0, 0.3, 1);
  }

  .intro.leaving {
    clip-path: inset(0 0 100% 0);
  }

  .stamp {
    opacity: 0;
    animation: stamp 420ms cubic-bezier(0.2, 1.8, 0.4, 1) forwards;
  }

  .stamp-1 {
    animation-delay: 150ms;
  }

  .stamp-2 {
    animation-delay: 500ms;
  }

  .stamp-3 {
    opacity: 0;
    animation: stamp 300ms cubic-bezier(0.2, 1.8, 0.4, 1) 950ms forwards;
  }

  @keyframes stamp {
    from {
      opacity: 0;
      transform: scale(2.4) rotate(-6deg);
    }
    to {
      opacity: 1;
      transform: scale(1) rotate(0);
    }
  }
</style>
