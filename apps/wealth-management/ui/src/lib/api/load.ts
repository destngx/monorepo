import { error } from '@sveltejs/kit';
import { toasts } from '$lib/state/toasts.svelte';
import { ApiError, type SheetsAuthKind } from './client';

/**
 * Why a request failed, for the message and the way out:
 * - `auth`: Google rejected the Sheets token, or the API token is wrong
 * - `not-configured`: Google Sheets is not set up on the engine
 * - `unavailable`: an optional service (AI, market data) is off or down
 * - `unreachable`: the engine did not answer
 * - `other`: anything else; the message says what
 */
export type FailureKind = 'auth' | 'not-configured' | 'unavailable' | 'unreachable' | 'other';

export interface LoadFailure {
  kind: FailureKind;
  message: string;
  /** Set for Google Sheets credential problems: which steps fix them (WM-UI-4). */
  sheetsAuth: SheetsAuthKind | null;
  /** What to do, when the engine names an optional service that is off. */
  hint: string | null;
  status?: number;
}

const LEGACY_MISSING = 'This data is unavailable: Google Sheets is not configured on the engine.';

/** Sorts a failure into the kinds above; never throws. */
export function classifyFailure(cause: unknown): LoadFailure {
  if (cause instanceof ApiError) {
    // Engines without the error envelope answer 404 when the sheet is not
    // configured; newer ones answer 503 SHEETS_NOT_CONFIGURED.
    if (cause.status === 404 && cause.code === undefined) {
      return { kind: 'not-configured', message: LEGACY_MISSING, sheetsAuth: 'missing', hint: null, status: 503 };
    }
    const sheetsAuth = cause.sheetsAuth;
    const base = { message: cause.message, sheetsAuth, hint: null, status: cause.status };
    if (sheetsAuth === 'missing') return { ...base, kind: 'not-configured' };
    if (sheetsAuth === 'expired' || cause.status === 401 || cause.status === 403) return { ...base, kind: 'auth' };
    if (cause.code === 'UNAVAILABLE') return { ...base, kind: 'unavailable', hint: cause.unavailableHint };
    return { ...base, kind: 'other' };
  }
  if (cause instanceof TypeError) {
    return { kind: 'unreachable', message: `Engine unreachable: ${cause.message}`, sheetsAuth: null, hint: null };
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
        authProblem: failure.sheetsAuth !== null,
        sheetsAuth: failure.sheetsAuth ?? undefined,
      });
    }
    error(502, { message: cause instanceof Error ? `Engine unreachable: ${cause.message}` : 'Engine unreachable' });
  }
}

/**
 * A page part that may fail on its own: its value, or the error to show in
 * its place. `failure` is set by `defer` (older `settle` callers keep `error`).
 */
export type Settled<T> = { ok: true; value: T } | { ok: false; error: string; failure?: LoadFailure };

/** What a load function returns for a part that streams in: it never rejects. */
export type Streamed<T> = Promise<Settled<T>>;

/**
 * Loads a part of a page whose failure must not fail the page (the home
 * inbox, safe-to-spend): the error is kept and shown where the part goes.
 */
export async function settle<T>(promise: Promise<T>): Promise<Settled<T>> {
  try {
    return { ok: true, value: await promise };
  } catch (cause) {
    return { ok: false, error: cause instanceof Error ? cause.message : String(cause) };
  }
}

/**
 * Starts a request without waiting for it, for a page section that streams
 * in (render it with <Deferred>). The promise never rejects: a failure
 * resolves to `{ ok: false, error, failure }` with the failure classified.
 * A Google Sheets credential failure also raises the layout's re-auth alert
 * (SheetsAuthAlert), since every ledger section of the page needs it.
 */
export function defer<T>(load: Promise<T> | (() => Promise<T>)): Streamed<T> {
  const fail = (cause: unknown): Settled<T> => {
    const failure = classifyFailure(cause);
    if (failure.sheetsAuth) toasts.sheetsAuthProblem = { message: failure.message, kind: failure.sheetsAuth };
    return { ok: false, error: failure.message, failure };
  };
  let promise: Promise<T>;
  try {
    promise = typeof load === 'function' ? load() : load;
  } catch (cause) {
    return Promise.resolve(fail(cause));
  }
  return promise.then((value): Settled<T> => ({ ok: true, value }), fail);
}

/** A settled result for a value already at hand (e.g. after a save returned the new figures). */
export function settled<T>(value: T): Streamed<T> {
  return Promise.resolve({ ok: true, value });
}
