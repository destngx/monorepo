import { error } from '@sveltejs/kit';
import { createLoader, UNREACHABLE_HINT, type LoadFailure as SharedFailure } from '@libs/ui-kit';
import { sheetsAuth } from '$lib/state/toasts.svelte';
import { ApiError, type SheetsAuthKind } from './client';

export { settle, settled, type FailureKind, type Settled, type Streamed } from '@libs/ui-kit';

/**
 * Why a request failed (ui-kit's LoadFailure), with this app's kinds:
 * - `auth`: Google rejected the Sheets token, or the API token is wrong
 * - `not-configured`: Google Sheets is not set up on the engine
 * - `unavailable`: an optional service (AI, market data) is off or down
 * - `unreachable`: the engine did not answer
 * - `other`: anything else; the message says what
 */
export interface LoadFailure extends SharedFailure {
  /** Set for Google Sheets credential problems: which steps fix them (WM-UI-4). */
  sheetsAuth?: SheetsAuthKind | null;
}

const LEGACY_MISSING = 'This data is unavailable: Google Sheets is not configured on the engine.';
const SHEETS_HINT = 'Google Sheets needs attention: follow the steps at the top of the page.';

/** Sorts a failure into the kinds above; never throws. */
export function classifyFailure(cause: unknown): LoadFailure {
  if (cause instanceof ApiError) {
    // Engines without the error envelope answer 404 when the sheet is not
    // configured; newer ones answer 503 SHEETS_NOT_CONFIGURED.
    if (cause.status === 404 && cause.code === undefined) {
      return { kind: 'not-configured', message: LEGACY_MISSING, sheetsAuth: 'missing', hint: SHEETS_HINT, status: 503 };
    }
    const sheetsAuth = cause.sheetsAuth;
    const base = { message: cause.message, sheetsAuth, hint: sheetsAuth ? SHEETS_HINT : null, status: cause.status };
    if (sheetsAuth === 'missing') return { ...base, kind: 'not-configured' };
    if (sheetsAuth === 'expired' || cause.status === 401 || cause.status === 403) return { ...base, kind: 'auth' };
    if (cause.code === 'UNAVAILABLE') return { ...base, kind: 'unavailable', hint: cause.unavailableHint };
    return { ...base, kind: 'other' };
  }
  if (cause instanceof TypeError) {
    return {
      kind: 'unreachable',
      message: `Engine unreachable: ${cause.message}`,
      sheetsAuth: null,
      hint: UNREACHABLE_HINT,
    };
  }
  const message = cause instanceof Error ? cause.message : String(cause);
  return { kind: 'other', message, sheetsAuth: null, hint: null };
}

/**
 * Runs page-load requests and turns API failures into SvelteKit errors, so
 * +error.svelte can show the server message (and the re-auth hint). Use it
 * only for data the page shell cannot render without; everything else is
 * `defer`red.
 */
export async function loadOrError<T>(load: () => Promise<T>): Promise<T> {
  try {
    return await load();
  } catch (cause) {
    if (cause instanceof ApiError) {
      const failure = classifyFailure(cause);
      error(failure.status ?? cause.status, {
        message: failure.message,
        status: failure.status ?? cause.status,
        authProblem: !!failure.sheetsAuth,
        sheetsAuth: failure.sheetsAuth ?? undefined,
      });
    }
    error(502, { message: cause instanceof Error ? `Engine unreachable: ${cause.message}` : 'Engine unreachable' });
  }
}

const loader = createLoader<LoadFailure>({
  classify: classifyFailure,
  // Every ledger section of the page needs Sheets, so a credential failure also raises the
  // layout's re-auth alert (SheetsAuthAlert).
  onFailure: (failure) => {
    if (failure.sheetsAuth) sheetsAuth.problem = { message: failure.message, kind: failure.sheetsAuth };
  },
});

/**
 * Starts a request without waiting for it, for a page section that streams
 * in (render it with <Deferred>). The promise never rejects: a failure
 * resolves to `{ ok: false, error, failure }` with the failure classified.
 */
export const defer = loader.defer;
