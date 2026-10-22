import { createTheme } from '@libs/ui-kit';

export type { Theme } from '@libs/ui-kit';

/** The theme setting; app.html applies the saved one before the first paint. */
export const theme = createTheme('hw.theme');
