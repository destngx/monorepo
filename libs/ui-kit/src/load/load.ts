/**
 * Why a request failed, for the message and the way out:
 * - `auth`: the engine or a provider rejected the credentials (401, 403)
 * - `not-configured`: a needed integration is not set up
 * - `unavailable`: an optional service (AI, market data) is off or down
 * - `unreachable`: the engine did not answer
 * - `other`: anything else; the message says what
 */
export type FailureKind = 'auth' | 'not-configured' | 'unavailable' | 'unreachable' | 'other';

export interface LoadFailure {
  kind: FailureKind;
  message: string;
  /** What to do about it, shown under the message. */
  hint: string | null;
  status?: number;
}

/** Sorts a thrown value into a LoadFailure; it must never throw. */
export type Classify = (cause: unknown) => LoadFailure;

export const UNREACHABLE_HINT = 'Check that the engine is running, then retry.';

/** The fields this library reads from an API error: the HTTP status and the error envelope's code. */
interface HttpError {
  message: string;
  status: number;
  code?: unknown;
}

function isHttpError(cause: unknown): cause is HttpError {
  return cause instanceof Error && typeof (cause as Partial<HttpError>).status === 'number';
}

/**
 * The default classifier, for any Error with a numeric `status` and an
 * optional string `code` (the engines' ApiError). A fetch that never reached
 * the engine throws a TypeError.
 */
export function classifyFailure(cause: unknown): LoadFailure {
  if (isHttpError(cause)) {
    const base = { message: cause.message, hint: null, status: cause.status };
    const code = typeof cause.code === 'string' ? cause.code : '';
    if (code === 'NOT_CONFIGURED' || code.endsWith('_NOT_CONFIGURED')) return { ...base, kind: 'not-configured' };
    if (cause.status === 401 || cause.status === 403) return { ...base, kind: 'auth' };
    if (code === 'UNAVAILABLE') return { ...base, kind: 'unavailable' };
    return { ...base, kind: 'other' };
  }
  if (cause instanceof TypeError) {
    return { kind: 'unreachable', message: `Engine unreachable: ${cause.message}`, hint: UNREACHABLE_HINT };
  }
  return { kind: 'other', message: cause instanceof Error ? cause.message : String(cause), hint: null };
}

/** A page part that may fail on its own: its value, or the error to show in its place. */
export type Settled<T> = { ok: true; value: T } | { ok: false; error: string; failure?: LoadFailure };

/** What a load function returns for a part that streams in: it never rejects. */
export type Streamed<T> = Promise<Settled<T>>;

/** Waits for a part whose failure must not fail the page, keeping the error to show where the part goes. */
export async function settle<T>(promise: Promise<T>): Promise<Settled<T>> {
  try {
    return { ok: true, value: await promise };
  } catch (cause) {
    return { ok: false, error: cause instanceof Error ? cause.message : String(cause) };
  }
}

/** A settled result for a value already at hand (e.g. after a save returned the new figures). */
export function settled<T>(value: T): Streamed<T> {
  return Promise.resolve({ ok: true, value });
}

export interface LoaderOptions {
  /** The app's own classifier; defaults to classifyFailure. */
  classify?: Classify;
  /** Runs for every failure, e.g. to raise an alert the whole page needs. */
  onFailure?: (failure: LoadFailure, cause: unknown) => void;
}

export interface Loader {
  classify: Classify;
  defer: <T>(load: Promise<T> | (() => Promise<T>)) => Streamed<T>;
}

/**
 * A `defer` that uses the app's classifier and failure hook. An app makes one
 * in its `api/load.ts` and uses it in every load function.
 */
export function createLoader(options: LoaderOptions = {}): Loader {
  const classify = options.classify ?? classifyFailure;
  return {
    classify,
    defer: <T>(load: Promise<T> | (() => Promise<T>)): Streamed<T> => {
      const fail = (cause: unknown): Settled<T> => {
        const failure = classify(cause);
        options.onFailure?.(failure, cause);
        return { ok: false, error: failure.message, failure };
      };
      let promise: Promise<T>;
      try {
        promise = typeof load === 'function' ? load() : load;
      } catch (cause) {
        return Promise.resolve(fail(cause));
      }
      return promise.then((value): Settled<T> => ({ ok: true, value }), fail);
    },
  };
}

/**
 * Starts a request without waiting for it, for a page section that streams
 * in (render it with <Deferred>). The promise never rejects: a failure
 * resolves to `{ ok: false, error, failure }` with the failure classified.
 */
export const defer = createLoader().defer;
