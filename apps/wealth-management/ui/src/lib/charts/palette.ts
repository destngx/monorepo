// The series order (PALETTE), PRIMARY and NEUTRAL come from ui-kit; the domain pairs below are
// validated on this app's surfaces, #fdfbf7 and #1c1916 (--color-surface in routes/layout.css).
import { NEUTRAL, PRIMARY } from '@libs/ui-kit';

export { NEUTRAL, PALETTE, PRIMARY } from '@libs/ui-kit';

// A third series beside INCOME and EXPENSE (the card cashback estimate); the three pass as a set.
export const BESIDE_FLOWS = '#7a5da9';

// Money in and money out; validated as a pair on both themes.
export const INCOME = '#018e7d';
export const EXPENSE = '#cd6a39';
// Needs vs wants (WM-DASH-8); validated as a set on both themes. Untagged is a neutral, always labelled.
export const FIXED = '#376daf';
export const ESSENTIAL = '#017c5d';
export const FLEXIBLE = '#bc822e';
export const UNTAGGED = NEUTRAL;
// Net worth projection, nominal vs real (after inflation); validated as a pair on both themes.
export const NOMINAL = '#376daf';
export const REAL = '#017c5d';
// Gold and USD protection trends (ROADMAP M4); validated as a pair on both themes.
export const GOLD = '#bc822e';
export const USD = '#2590b2';
// Stacked spend (WM-BUD-12, WM-CAT-3): day-to-day spend under fixed costs or a split tag; validated
// as a pair on both themes.
export const DAY_TO_DAY = '#ea580c';
export const SPLIT_SHARE = '#376daf';
// A second split tag (more fold into it as "Other split tags"); the three pass as a set on both themes.
export const SPLIT_SECOND = '#017c5d';
// Net worth history (ROADMAP P3): one series, the first categorical hue. Estimated months are told
// apart by a dashed line and hollow markers, milestones by a labelled diamond, never by colour.
export const NET_WORTH = PRIMARY;
