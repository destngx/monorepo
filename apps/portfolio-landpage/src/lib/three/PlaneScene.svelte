<script lang="ts">
  import { T, useTask, useThrelte } from '@threlte/core';
  import { untrack } from 'svelte';
  import { AnimationMixer, Group, LoopRepeat, MathUtils } from 'three';
  import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
  import { fitToSize, toonify } from './toon';

  type Props = { pointer: { x: number; y: number }; isPaused: boolean; ink: string };

  let { pointer, isPaused, ink }: Props = $props();

  const { invalidate, renderMode } = useThrelte();
  // The airframe. The rest of the model is a smoke-trail rig that spans roughly 3x the plane, so framing the
  // whole scene would shrink the plane to a speck.
  const PLANE_BODY_NODE = 'polySurface172';
  const PLANE_SIZE = 4.2;
  const OUTLINE_RATIO = 0.008;
  const rig = new Group();
  let model = $state<Group>();
  let mixer: AnimationMixer | undefined;
  let elapsed = 0;

  $effect(() => {
    let isCancelled = false;
    new GLTFLoader()
      .loadAsync('/models/plane.glb')
      .then((gltf) => {
        if (isCancelled) return;
        const scene = gltf.scene;
        fitToSize(scene, PLANE_SIZE, scene.getObjectByName(PLANE_BODY_NODE) ?? scene);
        toonify(scene, { ink: untrack(() => ink), outlineThickness: PLANE_SIZE * OUTLINE_RATIO });
        if (gltf.animations.length) {
          mixer = new AnimationMixer(scene);
          for (const clip of gltf.animations) mixer.clipAction(clip).setLoop(LoopRepeat, Infinity).play();
        }
        model = scene;
      })
      .catch((error: unknown) => {
        // Decorative only: the hero still works without the plane.
        console.error('Failed to load plane model', error);
      });
    return () => {
      isCancelled = true;
      mixer?.stopAllAction();
    };
  });

  // Recolour the ink outline when the theme flips.
  $effect(() => {
    if (model) {
      toonify(model, { ink });
      invalidate();
    }
  });

  // Resting yaw: a three-quarter profile. The model is authored nose-forward along Z, which reads as a speck head-on.
  const BASE_YAW = -1.05;

  const { start, stop } = useTask((delta) => {
    elapsed += delta;
    mixer?.update(delta);
    rig.position.y = Math.sin(elapsed * 1.6) * 0.18;
    rig.rotation.y = MathUtils.lerp(rig.rotation.y, BASE_YAW + pointer.x * 0.5, 0.05);
    rig.rotation.z = MathUtils.lerp(rig.rotation.z, -pointer.x * 0.35 + Math.sin(elapsed) * 0.05, 0.05);
    rig.rotation.x = MathUtils.lerp(rig.rotation.x, pointer.y * 0.25, 0.05);
  });

  $effect(() => {
    renderMode.set(isPaused ? 'manual' : 'always');
    if (isPaused) stop();
    else start();
  });
</script>

<T.PerspectiveCamera makeDefault position={[0, 0.6, 7]} fov={40} oncreate={(camera) => camera.lookAt(0, 0, 0)} />
<T.AmbientLight intensity={1.4} />
<T.DirectionalLight position={[4, 6, 5]} intensity={2.4} />

<T is={rig}>
  {#if model}
    <T is={model} />
  {/if}
</T>
