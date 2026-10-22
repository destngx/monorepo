import { leftColorScale } from '@libs/svelte-ai-usage';
import { STATUS } from '$lib/status';

// Presentation of the OpenAI quota reported by the ai-gateway (WM-SET-4); the
// rest is shared with health-watcher in @libs/svelte-ai-usage.

/** Fill colour: amber from 70% used, rose from 90%. */
export function usageColor(usedPercent: number): string {
  if (usedPercent >= 90) return STATUS.critical.fill;
  if (usedPercent >= 70) return STATUS.warning.fill;
  return STATUS.good.fill;
}

/** Green (all left) to red (none left) through amber, light enough to read on light and dark surfaces. */
export const leftColor = leftColorScale({ lightness: 0.7, chroma: 0.17, emptyHue: 25, fullHue: 150 });
