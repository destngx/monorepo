import type { IconName } from './components/Icon.svelte';

/**
 * One status scale for every page. A status is never shown by colour alone:
 * render its icon and a label with it (Badge `status`, or icon and text).
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
  /** Tag background and text (Badge). */
  badge: string;
  /** Solid fill for bars and dots. */
  fill: string;
  /** Light fill for table cells and heatmaps. */
  wash: string;
  /** Chart colour (SVG fill or stroke); readable on the light and the dark surface. */
  chart: string;
}

/*
 * The classes use the theme's role scales (theme.css), so an app's palette
 * carries through. Serious and critical share the danger hue at two
 * strengths; their icons and labels tell them apart.
 *
 * Chart colours are wealth-management's, each passing the dataviz palette
 * check on both themes (OKLCH lightness band, chroma floor, 3:1 contrast on
 * the light and dark surfaces). Red and green are not told apart by
 * colour-blind readers, so a status mark always carries its icon and label.
 */
export const STATUS: Record<StatusTone, StatusToken> = {
  good: {
    icon: 'check',
    label: 'Good',
    ink: 'text-good-700 dark:text-good-300',
    edge: 'border-l-good-500',
    box: 'border-good-500/30 bg-good-500/5 text-good-700 dark:text-good-300',
    badge: 'bg-good-500/15 text-good-700 dark:text-good-300',
    fill: 'bg-good-500',
    wash: 'bg-good-500/15',
    chart: '#059669',
  },
  warning: {
    icon: 'alert',
    label: 'Warning',
    ink: 'text-warn-700 dark:text-warn-300',
    edge: 'border-l-warn-500',
    box: 'border-warn-500/30 bg-warn-500/5 text-warn-700 dark:text-warn-300',
    badge: 'bg-warn-500/15 text-warn-700 dark:text-warn-300',
    fill: 'bg-warn-400',
    wash: 'bg-warn-400/25',
    chart: '#d97706',
  },
  serious: {
    icon: 'alertCircle',
    label: 'Serious',
    ink: 'text-danger-600 dark:text-danger-300',
    edge: 'border-l-danger-400',
    box: 'border-danger-400/30 bg-danger-400/5 text-danger-600 dark:text-danger-300',
    badge: 'bg-danger-400/15 text-danger-600 dark:text-danger-300',
    fill: 'bg-danger-400',
    wash: 'bg-danger-400/20',
    chart: '#ea580c',
  },
  critical: {
    icon: 'octagon',
    label: 'Critical',
    ink: 'text-danger-700 dark:text-danger-200',
    edge: 'border-l-danger-600',
    box: 'border-danger-600/40 bg-danger-600/10 text-danger-700 dark:text-danger-200',
    badge: 'bg-danger-600/20 text-danger-700 dark:text-danger-200',
    fill: 'bg-danger-600',
    wash: 'bg-danger-600/25',
    chart: '#e11d48',
  },
  info: {
    icon: 'info',
    label: 'Info',
    ink: 'text-info-700 dark:text-info-300',
    edge: 'border-l-info-500',
    box: 'border-info-500/30 bg-info-500/5 text-info-700 dark:text-info-300',
    badge: 'bg-info-500/15 text-info-700 dark:text-info-300',
    fill: 'bg-info-500',
    wash: 'bg-info-500/15',
    chart: '#0284c7',
  },
  neutral: {
    icon: 'info',
    label: 'No status',
    ink: 'text-ink-700 dark:text-ink-300',
    edge: 'border-l-ink-400',
    box: 'border-ink-400/30 bg-ink-500/5 text-ink-700 dark:text-ink-300',
    badge: 'bg-ink-900/5 text-ink-700 dark:bg-ink-50/10 dark:text-ink-300',
    fill: 'bg-ink-400',
    wash: 'bg-ink-500/10',
    chart: '#64748b',
  },
};

/** Share of a limit used: good below `warnAt`, warning from it, serious at 100% and over. */
export function usageTone(percent: number, warnAt = 80): StatusTone {
  if (percent >= 100) return 'serious';
  if (percent >= warnAt) return 'warning';
  return 'good';
}
