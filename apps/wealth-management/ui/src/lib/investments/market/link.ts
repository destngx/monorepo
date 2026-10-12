import { base } from '$app/paths';
import type { Timeframe } from '$lib/api/types';
import { analysisHref } from '../market';

/** App path to a symbol's analysis page on the given timeframe. */
export const symbolHref = (symbol: string, timeframe: Timeframe) => `${base}${analysisHref(symbol, timeframe)}`;
