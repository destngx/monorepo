<script lang="ts">
  import { onMount } from 'svelte';
  import { prefersReducedMotion } from 'svelte/motion';
  import type { Component } from 'svelte';
  import { profile } from '$lib/data/profile';

  let PlaneCanvas = $state<Component<{ ink: string }>>();
  let ink = $state('#121110');

  const readInk = () => getComputedStyle(document.documentElement).getPropertyValue('--ink').trim() || '#121110';

  const supportsWebGL = () => {
    try {
      return !!document.createElement('canvas').getContext('webgl2');
    } catch {
      return false;
    }
  };

  onMount(() => {
    ink = readInk();
    const observer = new MutationObserver(() => (ink = readInk()));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });

    // The 3D plane is a progressive enhancement: lazy-loaded, skipped for reduced motion or no WebGL.
    if (!prefersReducedMotion.current && supportsWebGL()) {
      import('$lib/three/PlaneCanvas.svelte')
        .then((module) => (PlaneCanvas = module.default))
        .catch((error: unknown) => console.error('Failed to load 3D scene', error));
    }

    return () => observer.disconnect();
  });
</script>

<section
  class="relative grid items-center gap-12 pt-12 pb-20 md:grid-cols-[1.15fr_1fr] md:pt-20"
  aria-labelledby="hero-title"
>
  <div class="relative z-10">
    <p class="bubble hero-pop inline-block font-display text-xl md:text-2xl">{profile.greeting}</p>

    <h1
      id="hero-title"
      class="hero-pop mt-10 font-display text-[2.6rem] leading-[1.02] tracking-tight md:text-6xl lg:text-7xl"
      style="--pop-delay: 120ms"
    >
      I keep clusters <span class="marker">calm</span> and build odd little tools.
    </h1>

    <p class="hero-pop mt-6 max-w-xl text-lg text-ink-soft md:text-xl" style="--pop-delay: 240ms">
      {profile.name}, {profile.role.toLowerCase()} in {profile.location}. {profile.intro}
    </p>

    <div class="hero-pop mt-10 flex flex-wrap gap-4" style="--pop-delay: 360ms">
      <a class="btn btn-primary" href="#work">See the work</a>
      <a class="btn btn-ghost" href="#contact">Say hi</a>
    </div>
  </div>

  <div class="relative mx-auto w-full max-w-md">
    <div class="speedlines absolute -inset-16 -z-10 opacity-30" aria-hidden="true"></div>
    <figure class="panel hero-pop rotate-2 overflow-hidden" style="--pop-delay: 200ms">
      <img
        src="/images/avatar.jpg"
        alt="Manga-style ink portrait of Dinh wearing round glasses"
        width="800"
        height="800"
        class="aspect-square w-full object-cover grayscale-[0.1]"
        fetchpriority="high"
      />
      <figcaption
        class="absolute bottom-0 left-0 border-t-[3px] border-r-[3px] border-rule bg-panel px-3 py-1 font-mono text-xs font-semibold uppercase"
      >
        Vol. 1 / {profile.handle}
      </figcaption>
    </figure>
    <span class="sfx absolute -bottom-6 -left-8 -rotate-12 text-6xl md:text-7xl" aria-hidden="true">ゴゴゴ</span>

    {#if PlaneCanvas}
      <div
        class="plane pointer-events-none absolute -top-28 -right-16 h-60 w-72 md:-top-48 md:-right-28 md:h-80 md:w-[26rem]"
      >
        <PlaneCanvas {ink} />
      </div>
    {/if}
  </div>
</section>

<style>
  .marker {
    /* A thick underline raised into the glyphs: anchored to the font baseline, so it sits the same at every size. */
    text-decoration-line: underline;
    text-decoration-color: color-mix(in oklab, var(--accent) 60%, transparent);
    text-decoration-thickness: 0.3em;
    text-underline-offset: -0.22em;
    text-decoration-skip-ink: none;
  }

  .hero-pop {
    animation: pop 600ms cubic-bezier(0.2, 1.4, 0.4, 1) var(--pop-delay, 0ms) both;
  }

  .plane {
    animation: fly-in 1.4s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  }

  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(24px) scale(0.96);
    }
  }

  /* No scale here: the canvas sizes itself from this box, and a transformed box would be measured too small. */
  @keyframes fly-in {
    from {
      opacity: 0;
      transform: translate(-40%, 30%);
    }
  }
</style>
