<script lang="ts">
  import type { InvestmentPhase } from '$lib/api/types';
  import Icon, { type IconName } from '$lib/components/Icon.svelte';
  import { PHASES, type PhaseState } from '../thinktank';

  // Context → Think tank → Synthesis → Actions (WM-AI-9). A phase that needed
  // its fallback prompt shows amber with the reason the main prompt failed.
  let { phases }: { phases: Record<InvestmentPhase, PhaseState> } = $props();

  function look(phase: PhaseState): { icon: IconName; tone: string; label: string } {
    switch (phase.status) {
      case 'running':
        return {
          icon: 'clock',
          tone: 'border-sky-500/50 text-sky-700 dark:text-sky-300',
          label: phase.fallback ? 'Retrying with fallback…' : 'Running…',
        };
      case 'ok':
        return phase.fallback
          ? {
              icon: 'alert',
              tone: 'border-amber-500/50 text-amber-700 dark:text-amber-300',
              label: 'Done with fallback',
            }
          : { icon: 'check', tone: 'border-emerald-500/50 text-emerald-700 dark:text-emerald-300', label: 'Done' };
      case 'error':
        return { icon: 'x', tone: 'border-rose-500/50 text-rose-700 dark:text-rose-300', label: 'Failed' };
      case 'stopped':
        return { icon: 'stop', tone: 'border-slate-400/50 muted', label: 'Stopped' };
      default:
        return { icon: 'clock', tone: 'border-slate-900/10 dark:border-white/10 muted', label: 'Waiting' };
    }
  }
</script>

<ol class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4" aria-label="Analysis phases">
  {#each PHASES as { key, label }, i (key)}
    {@const phase = phases[key]}
    {@const view = look(phase)}
    <li class="rounded-xl border bg-white/40 px-3 py-2 dark:bg-white/5 {view.tone}">
      <p class="flex items-center gap-2 text-sm font-medium">
        <span
          class="grid size-6 shrink-0 place-items-center rounded-full bg-current/10 {phase.status === 'running'
            ? 'animate-pulse'
            : ''}"
        >
          <Icon name={view.icon} size={14} />
        </span>
        <span class="text-slate-800 dark:text-slate-100"><span class="muted">{i + 1}.</span> {label}</span>
      </p>
      <p class="mt-1 text-xs">{view.label}</p>
      {#if phase.fallback}
        <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">Main prompt failed: {phase.fallback}</p>
      {/if}
      {#if phase.status === 'error' && phase.error}
        <p class="mt-1 text-xs text-rose-700 dark:text-rose-300">{phase.error}</p>
      {/if}
    </li>
  {/each}
</ol>
