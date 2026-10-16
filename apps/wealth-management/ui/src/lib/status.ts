import type { IconName } from '$lib/components/Icon.svelte';

/**
 * One status scale for every page. A status is never shown by colour alone:
 * render its icon and a label with it (Badge `status`, StatusNote, or icon + text).
 */
export type StatusTone = 'good' | 'warning' | 'serious' | 'critical' | 'info' | 'neutral';

export const STATUS_TONES: readonly StatusTone[] = ['good', 'warning', 'serious', 'critical', 'info', 'neutral'];

export interface StatusToken {
  /** Icon shown next to the colour. */
  icon: IconName;
  /** Default label when the caller has no more specific one. */
  label: string;
  /** Text colour for a label or heading. */
  ink: string;
  /** Left accent border for a headline section. */
  edge: string;
  /** Tinted box: border, background and text. */
  box: string;
  /** Pill background and text (Badge). */
  badge: string;
  /** Solid fill for bars and dots. */
  fill: string;
  /** Light fill for table cells and heatmaps. */
  wash: string;
  /** Chart colour (SVG fill or stroke); readable on the light and the dark surface. */
  chart: string;
}

/*
 * Chart colours: each passes the dataviz palette check on both themes (OKLCH lightness band,
 * chroma floor, 3:1 contrast on #fcfcfb and #1a1a19). They are a status scale, not a series
 * palette: red and green are not told apart by colour-blind readers, so a status mark always
 * carries its icon and label. Neutral is a deliberate grey.
 */
export const STATUS: Record<StatusTone, StatusToken> = {
  good: {
    icon: 'check',
    label: 'Good',
    ink: 'text-emerald-700 dark:text-emerald-300',
    edge: 'border-l-emerald-500',
    box: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-300',
    badge: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
    fill: 'bg-emerald-500',
    wash: 'bg-emerald-500/15',
    chart: '#059669',
  },
  warning: {
    icon: 'alert',
    label: 'Warning',
    ink: 'text-amber-700 dark:text-amber-300',
    edge: 'border-l-amber-500',
    box: 'border-amber-500/30 bg-amber-500/5 text-amber-700 dark:text-amber-300',
    badge: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    fill: 'bg-amber-400',
    wash: 'bg-amber-400/25',
    chart: '#d97706',
  },
  serious: {
    icon: 'alertCircle',
    label: 'Serious',
    ink: 'text-orange-700 dark:text-orange-300',
    edge: 'border-l-orange-500',
    box: 'border-orange-500/30 bg-orange-500/5 text-orange-700 dark:text-orange-300',
    badge: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
    fill: 'bg-orange-500',
    wash: 'bg-orange-500/25',
    chart: '#ea580c',
  },
  critical: {
    icon: 'octagon',
    label: 'Critical',
    ink: 'text-rose-700 dark:text-rose-300',
    edge: 'border-l-rose-500',
    box: 'border-rose-500/30 bg-rose-500/5 text-rose-700 dark:text-rose-300',
    badge: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
    fill: 'bg-rose-500',
    wash: 'bg-rose-500/25',
    chart: '#e11d48',
  },
  info: {
    icon: 'info',
    label: 'Info',
    ink: 'text-sky-700 dark:text-sky-300',
    edge: 'border-l-sky-500',
    box: 'border-sky-500/30 bg-sky-500/5 text-sky-700 dark:text-sky-300',
    badge: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
    fill: 'bg-sky-500',
    wash: 'bg-sky-500/15',
    chart: '#0284c7',
  },
  neutral: {
    icon: 'info',
    label: 'No status',
    ink: 'text-slate-700 dark:text-slate-300',
    edge: 'border-l-slate-400',
    box: 'border-slate-400/30 bg-slate-500/5 text-slate-700 dark:text-slate-300',
    badge: 'bg-slate-900/5 text-slate-700 dark:bg-white/10 dark:text-slate-300',
    fill: 'bg-slate-400',
    wash: 'bg-slate-500/10',
    chart: '#64748b',
  },
};

/** Share of a limit used: good below `warnAt`, warning from it, serious at 100% and over. */
export function usageTone(percent: number, warnAt = 80): StatusTone {
  if (percent >= 100) return 'serious';
  if (percent >= warnAt) return 'warning';
  return 'good';
}
