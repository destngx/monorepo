import { PRIMARY } from '@libs/ui-kit';

// Chart colours. Checked with the dataviz palette validator as a pair on both
// theme surfaces, #fffcf6 (light) and #191916 (dark): lightness band, chroma
// floor, CVD and normal-vision separation, 3:1 contrast. Flags are also told
// apart by marker shape, never by colour alone.

/** A metric's one series. */
export const SERIES = PRIMARY;
/** A point the paper flagged (H, L or *); the same hue as danger-500. */
export const FLAGGED = '#a64f38';
