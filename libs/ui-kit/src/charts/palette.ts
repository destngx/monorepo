// Series colours shared by the charts, in a fixed order: muted navy, bronze, steel, rose, violet,
// olive, pine, mauve. The eight pass the dataviz palette check (lightness band, chroma floor, CVD
// and normal-vision separation of neighbours, 3:1 contrast) on both apps' light and dark
// surfaces: #fdfbf7 and #1c1916 (wealth-management), #fffcf6 and #191916 (health-watcher, the
// theme.css default). The ninth is a neutral for the rest. A ninth series folds into "Other";
// never generate a hue. An app's semantic pairs (income and expense) live in the app, each
// validated as a set.
export const PALETTE: readonly string[] = [
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

/** A chart's one series when it has a single one. */
export const PRIMARY = '#376daf';

/** A reference line or a series that is context, not data (a cap, a limit). */
export const NEUTRAL = '#a8a29e';
