<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { sentimentTone } from '../market';

  // WM-RULE-9 fear/greed gauge (0-100) from VIX and VN-Index RSI.
  let { score, label }: { score: number; label: string } = $props();

  const position = $derived(Math.max(0, Math.min(100, score)));
</script>

<div>
  <div class="flex items-baseline justify-between gap-3">
    <span class="tabular text-2xl font-semibold">{score}</span>
    <Badge tone={sentimentTone(score)}>{label}</Badge>
  </div>
  <div
    class="relative mt-3 h-2.5 rounded-sm bg-linear-to-r from-rose-500 via-amber-400 to-emerald-500"
    role="meter"
    aria-label="Market sentiment"
    aria-valuemin={0}
    aria-valuemax={100}
    aria-valuenow={score}
    aria-valuetext="{score} - {label}"
  >
    <span
      class="absolute top-1/2 size-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-stone-900 shadow dark:border-stone-900 dark:bg-white"
      style="left: {position}%"
    ></span>
  </div>
  <div class="muted mt-1 flex justify-between text-xs"><span>Fear</span><span>Neutral</span><span>Greed</span></div>
</div>
