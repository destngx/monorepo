// Presentation of the AI provider's quota that ai-gateway reports, shared by
// the apps that show it in their menu bar.

/** How often the gauge refreshes the quota while the tab is visible. */
export const AI_USAGE_REFRESH_MS = 5 * 60 * 1000;

export interface AIUsageWindow {
  /** "5h" or "weekly". */
  name: string;
  usedPercent: number;
  resetsAt?: string;
}

/** The app's quota state the gauge reads and refreshes. */
export interface AIUsageSource {
  /** False hides the gauge, as while AI is switched off; absent means on. */
  readonly enabled?: boolean;
  readonly usage: { provider: string; windows: AIUsageWindow[] } | null;
  /** Why the quota could not be read; empty when it was. */
  readonly error: string;
  readonly loadedAt: Date;
  load(): Promise<void>;
}

/** Colour for a share left, 0-100. */
export type LeftColor = (left: number) => string;

/** A reset time as the app writes dates and times. */
export type FormatReset = (resetsAt: string) => string;

const windowLabels: Record<string, string> = { '5h': '5-hour window', weekly: 'Weekly window' };

export function windowLabel(window: Pick<AIUsageWindow, 'name'>): string {
  return windowLabels[window.name] ?? window.name;
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

/** Share of the window still available, 0-100. */
export function leftPercent(window: Pick<AIUsageWindow, 'usedPercent'>): number {
  const left = 100 - window.usedPercent;
  return Number.isFinite(left) ? Math.min(100, Math.max(0, left)) : 0;
}

/**
 * A colour for the share left, from emptyHue (none left) to fullHue (all
 * left), in OKLCH so the lightness stays even along the way. Each app picks
 * the lightness and chroma that suit its theme.
 */
export function leftColorScale(scale: {
  lightness: number;
  chroma: number;
  emptyHue: number;
  fullHue: number;
}): LeftColor {
  return (left) => {
    const share = Math.min(100, Math.max(0, left)) / 100;
    const hue = scale.emptyHue + share * (scale.fullHue - scale.emptyHue);
    return `oklch(${scale.lightness} ${scale.chroma} ${hue.toFixed(1)})`;
  };
}

/** The weekly and 5-hour windows the gauge shows; null when absent. */
export function gaugeWindows(windows: AIUsageWindow[]): {
  weekly: AIUsageWindow | null;
  fiveHour: AIUsageWindow | null;
} {
  return {
    weekly: windows.find((window) => window.name === 'weekly') ?? null,
    fiveHour: windows.find((window) => window.name === '5h') ?? null,
  };
}
