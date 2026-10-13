import type { AIUsageWindow } from '$lib/api/types';

// Presentation of the OpenAI quota reported by the ai-gateway (WM-SET-4).

const windowLabels: Record<string, string> = { '5h': '5-hour window', weekly: 'Weekly window' };

export function windowLabel(window: Pick<AIUsageWindow, 'name'>): string {
  return windowLabels[window.name] ?? window.name;
}

/** Fill colour: amber from 70% used, rose from 90%. */
export function usageColor(usedPercent: number): string {
  if (usedPercent >= 90) return 'bg-rose-500';
  if (usedPercent >= 70) return 'bg-amber-400';
  return 'bg-emerald-500';
}

/** "resets in 2d 3h", "resets in 45m", "reset due"; empty when unknown. */
export function resetsIn(resetsAt: string | undefined, now = new Date()): string {
  if (!resetsAt) return '';
  const at = new Date(resetsAt).getTime();
  if (Number.isNaN(at)) return '';
  const minutes = Math.ceil((at - now.getTime()) / 60_000);
  if (minutes <= 0) return 'reset due';
  const days = Math.floor(minutes / 1440);
  const hours = Math.floor((minutes % 1440) / 60);
  const rest = minutes % 60;
  const parts =
    days > 0
      ? [`${days}d`, hours > 0 ? `${hours}h` : '']
      : hours > 0
        ? [`${hours}h`, rest > 0 ? `${rest}m` : '']
        : [`${rest}m`];
  return `resets in ${parts.filter(Boolean).join(' ')}`;
}
