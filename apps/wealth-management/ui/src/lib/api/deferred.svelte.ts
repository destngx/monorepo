import { classifyFailure, type Streamed, type LoadFailure, type Settled } from './load';

export interface TrackOptions {
  /**
   * What the data is about (a month, a code). When it changes, the shown
   * value is dropped and the skeleton returns; otherwise a new promise for
   * the same thing (a sync, a retry) keeps the old value on screen.
   */
  scope?: () => unknown;
}

/**
 * The latest result of a `defer`red promise, as reactive state: create it
 * while a component initialises. A new promise from the source (after
 * `invalidate`, a sync or a retry) keeps the last value visible and sets
 * `refreshing`, so refreshes never flash a skeleton.
 */
export class Tracked<T> {
  /** The last result; null until the first promise settles (or after a scope change). */
  settled = $state<Settled<T> | null>(null);
  /** The last value that loaded, kept while refreshing and after a failed refresh. */
  latest = $state<{ value: T } | null>(null);
  pending = $state(true);
  #token = 0;
  #scope: unknown = undefined;
  #promise: Promise<unknown> | null = null;

  constructor(source: () => Streamed<T> | null | undefined, options: TrackOptions = {}) {
    $effect.pre(() => {
      const promise = source();
      const scope = options.scope?.();
      // The same promise again (a parent re-derived its data) changes nothing.
      if (!promise || (promise === this.#promise && scope === this.#scope)) return;
      this.#promise = promise;
      const token = ++this.#token;
      if (scope !== this.#scope) {
        this.#scope = scope;
        this.settled = null;
        this.latest = null;
      }
      this.pending = true;
      // A raw promise that rejects is converted too, so nothing leaks.
      void promise
        .catch((cause: unknown): Settled<T> => {
          const failure = classifyFailure(cause);
          return { ok: false, error: failure.message, failure };
        })
        .then((result) => {
          if (token !== this.#token) return;
          this.settled = result;
          if (result.ok) this.latest = { value: result.value };
          this.pending = false;
        });
    });
  }

  /** The loaded value (kept while refreshing), or undefined before the first load. */
  get value(): T | undefined {
    return this.latest?.value;
  }

  /** Nothing to show yet: render a skeleton. */
  get loading(): boolean {
    return this.pending && this.latest === null;
  }

  /** A newer promise is in flight while an older value is shown. */
  get refreshing(): boolean {
    return this.pending && this.latest !== null;
  }

  /** The failure of the last result, if it failed. */
  get failure(): LoadFailure | null {
    const result = this.settled;
    if (!result || result.ok) return null;
    return result.failure ?? classifyFailure(new Error(result.error));
  }
}

/** Tracks a deferred promise from the page data; see Tracked. */
export function track<T>(source: () => Streamed<T> | null | undefined, options?: TrackOptions): Tracked<T> {
  return new Tracked(source, options);
}
