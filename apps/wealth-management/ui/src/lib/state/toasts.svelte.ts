import { toasts, type ErrorToast } from '@libs/ui-kit';
import { ApiError, type SheetsAuthKind } from '$lib/api/client';

export { toasts, type Toast, type ToastKind } from '@libs/ui-kit';

class SheetsAuth {
  /** Set when Google Sheets credentials fail, to show the re-auth alert (WM-UI-4). */
  problem = $state<{ message: string; kind: SheetsAuthKind } | null>(null);
}

export const sheetsAuth = new SheetsAuth();

/**
 * How this app shows a failed action: a Sheets credential failure raises the
 * re-auth alert instead of a toast, and an optional service that is off is a
 * warning with what to do.
 */
export const wealthErrorToast: ErrorToast = (error, text) => {
  if (error instanceof ApiError) {
    if (error.sheetsAuth) {
      sheetsAuth.problem = { message: error.message, kind: error.sheetsAuth };
      return null;
    }
    if (error.unavailableHint) return { kind: 'warning', message: `${text} ${error.unavailableHint}` };
  }
  return { kind: 'error', message: text };
};

// Every module that reports errors imports toasts from here (the root layout does too), so the
// shared toasts use this app's rules before the first error.
toasts.setErrorToast(wealthErrorToast);
