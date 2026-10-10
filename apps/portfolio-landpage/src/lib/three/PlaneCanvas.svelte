<script lang="ts">
  import { Canvas } from '@threlte/core';
  import PlaneScene from './PlaneScene.svelte';

  type Props = { ink: string };

  let { ink }: Props = $props();

  let container = $state<HTMLDivElement>();
  let isVisible = $state(true);
  let pointer = $state({ x: 0, y: 0 });

  $effect(() => {
    if (!container) return;
    const observer = new IntersectionObserver(([entry]) => (isVisible = entry.isIntersecting));
    observer.observe(container);
    return () => observer.disconnect();
  });

  const onPointerMove = (event: PointerEvent) => {
    pointer = {
      x: (event.clientX / window.innerWidth) * 2 - 1,
      y: (event.clientY / window.innerHeight) * 2 - 1,
    };
  };
</script>

<svelte:window onpointermove={onPointerMove} />

<div bind:this={container} class="h-full w-full" aria-hidden="true">
  <Canvas dpr={Math.min(window.devicePixelRatio, 2)}>
    <PlaneScene {pointer} isPaused={!isVisible} {ink} />
  </Canvas>
</div>
