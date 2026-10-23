import type { CategoryMonths, ExpenseSplit, SavingsTarget, YearVerdict } from '$lib/api/types';
import { formatMonth, formatPercent } from '$lib/format';
import type { StatusTone } from '$lib/status';

/** The `year` URL parameter, when it is a four-digit year. */
export function parseYear(raw: string | null): number | undefined {
  return raw && /^\d{4}$/.test(raw) ? Number(raw) : undefined;
}

/** What the year is compared against: the same elapsed months of the year before (WM-DASH-8). */
export function comparisonSpan(year: number, monthsElapsed: number): string | null {
  const previous = year - 1;
  if (monthsElapsed <= 0) return null;
  if (monthsElapsed >= 12) return String(previous);
  const last = formatMonth(`${previous}-${String(monthsElapsed).padStart(2, '0')}`, true);
  return monthsElapsed === 1 ? `${last} ${previous}` : `Jan-${last} ${previous}`;
}

/** Per-month sums across category rows (12 months when there are no rows). */
export function columnTotals(rows: CategoryMonths[]): number[] {
  const width = rows[0]?.values.length ?? 12;
  return Array.from({ length: width }, (_, i) => rows.reduce((sum, row) => sum + (row.values[i] ?? 0), 0));
}

/** Status tones a verdict uses; styled by the shared status tokens ($lib/status). */
export type VerdictTone = Extract<StatusTone, 'good' | 'warning' | 'serious' | 'critical' | 'neutral'>;

export interface VerdictMessage {
  tone: VerdictTone;
  title: string;
  text: string;
  detail: string | null;
}

/** Plain-words verdict for the year (WM-DASH-8); no amounts, so it reads the same masked. */
export function verdictMessage(
  year: number,
  verdict: YearVerdict,
  target: SavingsTarget,
  monthsElapsed: number,
): VerdictMessage {
  const rate = formatPercent(verdict.savingsRate);
  const deficits = verdict.deficitMonths;
  const detail =
    monthsElapsed === 0
      ? null
      : deficits === 0
        ? 'Every month earned more than it spent.'
        : `${deficits} of ${monthsElapsed} months spent more than ${deficits === 1 ? 'it' : 'they'} earned.`;
  switch (verdict.status) {
    case 'onTarget':
      return {
        tone: 'good',
        title: 'On target',
        text: `You kept ${rate} of your income, at or above the ${target.target}% target.`,
        detail,
      };
    case 'healthy':
      return {
        tone: 'good',
        title: 'Healthy',
        text: `You kept ${rate} of your income. ${target.healthy}-${target.target}% is healthy; ${target.target}% is the target.`,
        detail,
      };
    case 'watch':
      return {
        tone: 'warning',
        title: 'Worth watching',
        text: `You kept only ${rate} of your income. Aim for at least ${target.healthy}%.`,
        detail,
      };
    case 'atRisk':
      if (verdict.savingsRate < 0) {
        return {
          tone: 'critical',
          title: 'Spending more than you earn',
          text: `You spent ${formatPercent(-verdict.savingsRate)} more than you earned.`,
          detail,
        };
      }
      return {
        tone: 'serious',
        title: 'At risk',
        text: `You kept only ${rate} of your income, so one surprise bill could push you into debt. Aim for at least ${target.healthy}%.`,
        detail,
      };
    default:
      return { tone: 'neutral', title: 'Not enough data', text: `No income recorded for ${year} yet.`, detail };
  }
}

/** What the year-end projection is averaged over. */
export function projectionBasis(basisMonths: number): string {
  if (basisMonths <= 0) return 'This month so far, repeated for the year. Rough until a month has closed.';
  const last = formatMonth(`2000-${String(basisMonths).padStart(2, '0')}`, true);
  return `Months so far, the rest at your ${basisMonths === 1 ? last : `Jan-${last}`} average.`;
}

/** Rates past this are drawn at the edge; a month with tiny income can show -900%. */
export const SAVINGS_RATE_LIMIT = 100;

/** Y domain in percent for the savings rate chart: zero, the target band and the data, in steps of 10. */
export function savingsRateDomain(rates: number[], target: number): [number, number] {
  const clamp = (value: number) => Math.max(-SAVINGS_RATE_LIMIT, Math.min(SAVINGS_RATE_LIMIT, value));
  const low = clamp(Math.min(0, ...rates));
  const high = clamp(Math.max(target + 10, ...rates));
  return [Math.floor(low / 10) * 10, Math.ceil(high / 10) * 10];
}

/**
 * How needs, wants and savings share income (D45); percentages only, so it
 * reads the same masked. Untagged spending counts as a need. Needs leave
 * room only when they stay within the needs limit and the savings target is
 * met; wants get what is left after both.
 */
export function expenseSplitMessage(split: ExpenseSplit): VerdictMessage {
  if (split.fixed.total + split.essential.total + split.flexible.total === 0) {
    return {
      tone: 'neutral',
      title: 'Tag your categories',
      text: 'Write fixed, essential or flexible in column B of the Categories tab to see where your money must go and where it can be saved.',
      detail: null,
    };
  }
  const spending = split.fixed.total + split.essential.total + split.flexible.total + split.untagged.total;
  const untagged = split.untaggedCategories.length;
  const detail =
    untagged === 0
      ? null
      : `${untagged} untagged ${untagged === 1 ? 'category' : 'categories'} (${formatPercent((split.untagged.total / spending) * 100)} of spending) ${untagged === 1 ? 'counts' : 'count'} as needs until tagged.`;
  const needs = split.needsShareOfIncome;
  const wants = split.wantsShareOfIncome;
  const saved = split.savingsShareOfIncome;
  if (needs === null || wants === null || saved === null) {
    return {
      tone: 'neutral',
      title: 'Needs vs wants',
      text: 'No income recorded, so there is no share to judge yet.',
      detail,
    };
  }
  const limit = split.needsShareComfortable;
  const target = split.savingsShareTarget;
  const shares = `Needs take ${formatPercent(needs)} of your income, wants ${formatPercent(wants)}, and you save ${formatPercent(saved)}.`;
  // What wants may take once needs are paid and the savings target is set aside.
  const wantsRoom = Math.max(0, 100 - target - needs);
  if (needs > limit) {
    return {
      tone: saved < 0 ? 'critical' : 'warning',
      title: 'Needs are heavy',
      text: `${shares} Needs are above the ${limit}% limit, so the ${target}% savings target needs a lower fixed cost or trimmed essentials, not only fewer wants.`,
      detail,
    };
  }
  if (saved < target) {
    return {
      tone: saved < 0 ? 'critical' : 'warning',
      title: 'Wants take the savings',
      text: `${shares} Needs are within ${limit}%, but saving is under the ${target}% target. Bring wants down to ${formatPercent(wantsRoom)} of income to reach it.`,
      detail,
    };
  }
  return {
    tone: 'good',
    title: 'Needs leave room',
    text: `${shares} Needs are within ${limit}% and saving meets the ${target}% target. Keep saving first: wants can grow only to ${formatPercent(wantsRoom)} of income.`,
    detail,
  };
}
