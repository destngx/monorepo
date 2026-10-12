<script lang="ts">
  import type { Snippet } from 'svelte';

  type Tone = 'neutral' | 'green' | 'orange' | 'amber' | 'red' | 'blue' | 'violet';
  let { tone = 'neutral', children }: { tone?: Tone; children: Snippet } = $props();

  const tones: Record<Tone, string> = {
    neutral: 'bg-slate-900/5 text-slate-700 dark:bg-white/10 dark:text-slate-300',
    green: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
    orange: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
    amber: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    red: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
    blue: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
    violet: 'bg-violet-500/15 text-violet-700 dark:text-violet-300',
  };
</script>

<span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap {tones[tone]}"
  >{@render children()}</span
>
