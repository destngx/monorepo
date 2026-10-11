<script lang="ts">
  import { reveal } from '$lib/motion/reveal';

  type Props = { number: string; title: string; kicker?: string; id?: string };

  let { number, title, kicker, id }: Props = $props();
</script>

<div class="mb-10 flex items-end gap-4" use:reveal>
  <span
    class="grid h-16 w-16 shrink-0 -rotate-6 place-items-center border-[3px] border-rule bg-ink font-display text-2xl text-paper shadow-ink-sm"
    aria-hidden="true">{number}</span
  >
  <div>
    <p class="chapter-label">Chapter {number}{kicker ? ` / ${kicker}` : ''}</p>
    <h2 {id} class="font-display text-3xl leading-tight md:text-5xl">{title}</h2>
  </div>
</div>
