<!--
  Floating record player. The SoundCloud iframe is only created on the first open, so
  visitors who never press play load nothing from SoundCloud. After that it stays
  mounted while hidden, so the music keeps going when the panel is closed.
-->
<script lang="ts">
  import { site } from '$lib/data/site';
  import { soundcloudEmbedUrl } from '$lib/soundcloud';

  const PANEL_ID = 'soundcloud-player';
  // The light theme accent: the iframe loads once, and this pink reads well on both themes.
  const src = soundcloudEmbedUrl(site.soundcloudPlaylist, { color: '#e2306c' });

  let isOpen = $state(false);
  let hasLoaded = $state(false);

  const toggle = () => {
    isOpen = !isOpen;
    hasLoaded = true;
  };

  const onKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && isOpen) isOpen = false;
  };
</script>

<svelte:window onkeydown={onKeydown} />

<div class="fixed bottom-4 left-4 z-[80] flex flex-col items-start gap-3">
  {#if hasLoaded}
    <div
      id={PANEL_ID}
      class="player panel h-80 w-[min(22rem,calc(100vw-2rem))] overflow-hidden"
      class:open={isOpen}
      inert={!isOpen}
    >
      <iframe title="SoundCloud playlist" class="h-full w-full" allow="autoplay" loading="lazy" {src}></iframe>
    </div>
  {/if}

  <button
    type="button"
    class="panel grid h-12 w-12 place-items-center rounded-full !shadow-ink-sm transition-transform hover:scale-110 active:scale-90"
    aria-label={isOpen ? 'Hide the music player' : 'Play some music'}
    aria-expanded={isOpen}
    aria-controls={hasLoaded ? PANEL_ID : undefined}
    data-cursor={isOpen ? 'hide' : 'play'}
    onclick={toggle}
  >
    <!-- vinyl record: always black, whatever the theme -->
    <svg viewBox="0 0 24 24" class="record h-8 w-8" class:spinning={hasLoaded} aria-hidden="true">
      <circle cx="12" cy="12" r="11" fill="#121110" />
      <circle cx="12" cy="12" r="8.5" fill="none" stroke="#5c5852" stroke-width="0.6" />
      <circle cx="12" cy="12" r="6" fill="none" stroke="#5c5852" stroke-width="0.6" />
      <circle cx="12" cy="12" r="3.5" fill="var(--accent)" />
      <circle cx="12" cy="12" r="1" fill="#121110" />
    </svg>
  </button>
</div>

<style>
  .player {
    transform-origin: bottom left;
    opacity: 0;
    transform: translateY(12px) scale(0.92) rotate(-2deg);
    visibility: hidden;
    transition:
      opacity 180ms ease,
      transform 220ms cubic-bezier(0.3, 1.6, 0.5, 1),
      visibility 0s linear 220ms;
  }

  .player.open {
    opacity: 1;
    transform: none;
    visibility: visible;
    transition-delay: 0s;
  }

  /* Spins once music has been started; the global reduced-motion rule stops it. */
  .record.spinning {
    animation: spin 2.4s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
