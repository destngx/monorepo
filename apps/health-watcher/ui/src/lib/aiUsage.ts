import type { AIUsageWindow } from './api/types';

// Presentation of the AI provider's quota that ai-gateway reports (HW-OPS-10).

const windowLabels: Record<string, string> = { '5h': '5-hour window', weekly: 'Weekly window' };

export function windowLabel(window: Pick<AIUsageWindow, 'name'>): string {
  return windowLabels[window.name] ?? window.name;
}

/** Share of the window still available, 0-100. */
export function leftPercent(window: Pick<AIUsageWindow, 'usedPercent'>): number {
  const left = 100 - window.usedPercent;
  return Number.isFinite(left) ? Math.min(100, Math.max(0, left)) : 0;
}

/** Green (all left) to red (none left) through amber, in OKLCH so the lightness stays even. */
export function leftColor(left: number): string {
  const share = Math.min(100, Math.max(0, left)) / 100;
  return `oklch(0.7 0.17 ${(25 + share * 125).toFixed(1)})`;
}

/** "in 2d 3h", "in 45m", "due now"; empty when unknown. */
export function resetsIn(resetsAt: string | undefined, now = new Date()): string {
  if (!resetsAt) return '';
  const at = new Date(resetsAt).getTime();
  if (Number.isNaN(at)) return '';
  const minutes = Math.ceil((at - now.getTime()) / 60_000);
  if (minutes <= 0) return 'due now';
  const days = Math.floor(minutes / 1440);
  const hours = Math.floor((minutes % 1440) / 60);
  const rest = minutes % 60;
  const parts =
    days > 0
      ? [`${days}d`, hours > 0 ? `${hours}h` : '']
      : hours > 0
        ? [`${hours}h`, rest > 0 ? `${rest}m` : '']
        : [`${rest}m`];
  return `in ${parts.filter(Boolean).join(' ')}`;
}

const resetFormat = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Ho_Chi_Minh',
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

/** A reset time as `22/10 22:04`, the engine's calendar; empty when unreadable. */
export function formatReset(resetsAt: string): string {
  const at = new Date(resetsAt);
  return Number.isNaN(at.getTime()) ? '' : resetFormat.format(at).replace(',', '');
}

/** The weekly and 5-hour windows the gauge shows; null when absent. */
export function gaugeWindows(windows: AIUsageWindow[]): {
  weekly: AIUsageWindow | null;
  fiveHour: AIUsageWindow | null;
} {
  return {
    weekly: windows.find((w) => w.name === 'weekly') ?? null,
    fiveHour: windows.find((w) => w.name === '5h') ?? null,
  };
}
