import { toasts } from './toasts.svelte';

export interface ActionOptions {
  /** Toast shown when the action finishes. */
  success?: string;
  /** Prefix of the error toast, e.g. "Could not save" (default "Action failed"). */
  errorContext?: string;
}

/**
 * An async action with a busy flag and toast feedback, for controls that are
 * not a <Button> (menus, rows, forms). `run` never rejects: a failure is
 * reported with toasts.error (Sheets credential failures raise the re-auth
 * alert) and `run` resolves to false. While busy, further runs are ignored.
 */
export class Action<A extends unknown[]> {
  busy = $state(false);
  readonly #fn: (...args: A) => unknown;
  readonly #options: ActionOptions;

  constructor(fn: (...args: A) => unknown, options: ActionOptions = {}) {
    this.#fn = fn;
    this.#options = options;
  }

  run = async (...args: A): Promise<boolean> => {
    if (this.busy) return false;
    this.busy = true;
    try {
      await this.#fn(...args);
      if (this.#options.success) toasts.success(this.#options.success);
      return true;
    } catch (error) {
      toasts.error(error, this.#options.errorContext ?? 'Action failed');
      return false;
    } finally {
      this.busy = false;
    }
  };
}

export function action<A extends unknown[]>(fn: (...args: A) => unknown, options?: ActionOptions): Action<A> {
  return new Action(fn, options);
}
