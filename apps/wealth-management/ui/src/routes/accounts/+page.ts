import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// 366 days covers the 12-month balance history (WM-ACC-8) and each account's sparkline (WM-ACC-3).
const HISTORY_DAYS = 366;

// Every request streams into its own section: the header and New account show
// at once, and a slow or failing call (the allocation values the portfolio)
// holds only its card.
export const load: PageLoad = ({ fetch }) => ({
  accounts: defer(() => api.accounts({ fetch })),
  histories: defer(() => api.accountHistories(HISTORY_DAYS, { fetch })),
  summary: defer(() => api.dashboard({ fetch })),
  allocation: defer(() => api.allocation({ fetch })),
  // The card set-up holds the bank limits that card accounts share (WM-CARD-16).
  cards: defer(() => api.cards({ fetch })),
  // Wallet earning balance rates (D41), and the idle cash links that tie an account to one.
  flexibleSavings: defer(() => api.flexibleSavings(fetch)),
  preferences: defer(() => api.preferences({ fetch })),
});
