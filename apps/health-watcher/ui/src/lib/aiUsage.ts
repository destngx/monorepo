import { leftColorScale } from '@libs/ui-kit';

// Presentation of the AI provider's quota that ai-gateway reports (HW-OPS-10);
// the rest is shared with wealth-management in @libs/ui-kit.

/** Forest green (all left) to oxblood (none left) through brass, muted to match the theme. */
export const leftColor = leftColorScale({ lightness: 0.58, chroma: 0.09, emptyHue: 30, fullHue: 145 });

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
