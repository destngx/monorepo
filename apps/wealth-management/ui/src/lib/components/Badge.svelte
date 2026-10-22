<script lang="ts">
  import type { Snippet } from 'svelte';
  import { STATUS, type StatusTone } from '$lib/status';
  import Icon from './Icon.svelte';

  // `tone` is a plain hue for categories and tags. A status uses `status`, which adds the status icon
  // and falls back to the status label, so it never reads by colour alone. The green / amber / red /
  // orange hues remain only for pages that have not moved to `status` yet.
  type Tone = 'neutral' | 'green' | 'orange' | 'amber' | 'red' | 'blue' | 'violet';
  let { tone = 'neutral', status, children }: { tone?: Tone; status?: StatusTone; children?: Snippet } = $props();

  const tones: Record<Tone, string> = {
    neutral: 'bg-stone-900/5 text-stone-700 dark:bg-white/10 dark:text-stone-300',
    green: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
    orange: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
    amber: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    red: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
    blue: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
    violet: 'bg-violet-500/15 text-violet-700 dark:text-violet-300',
  };
  const token = $derived(status ? STATUS[status] : null);
</script>

<span
  class="inline-flex items-center gap-1 rounded-md border border-current/25 px-2 py-0.5 text-xs font-medium whitespace-nowrap {token
    ? token.badge
    : tones[tone]}"
  data-status={status}
  >{#if token}<Icon
      name={token.icon}
      size={12}
    />{/if}{#if children}{@render children()}{:else if token}{token.label}{/if}</span
>
