// Series colours shared by the charts, in a fixed order: muted navy, bronze, steel, rose, violet,
// olive, pine, mauve. The eight pass the dataviz palette check (lightness band, chroma floor,
// CVD and normal-vision separation of neighbours, 3:1 contrast) on both theme surfaces, #fdfbf7
// and #1c1916 (--color-surface in routes/layout.css). The ninth is a neutral for the rest.
export const PALETTE = [
  '#376daf',
  '#bc822e',
  '#2590b2',
  '#b7535e',
  '#7a5da9',
  '#869638',
  '#017c5d',
  '#c576a4',
  '#78716c',
];

// A chart's one series when it has a single one.
export const PRIMARY = '#376daf';
// A reference line or a series that is context, not data (a cap, a limit).
export const NEUTRAL = '#a8a29e';
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
