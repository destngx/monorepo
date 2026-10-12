import { toasts } from '$lib/state/toasts.svelte';
import { ApiError } from '$lib/api/client';

export type TaskStatus = 'idle' | 'loading' | 'done' | 'error';

/**
 * One on-demand AI request with its UI state. A newer run aborts the older
 * one; failures stay visible with their message so the UI can offer a retry
 * instead of an empty card (WM-AI-15).
 */
export class AITask<T> {
  status = $state<TaskStatus>('idle');
  value = $state<T | undefined>(undefined);
  error = $state('');
  #controller: AbortController | null = null;
  #load: (force: boolean, signal: AbortSignal) => Promise<T>;

  constructor(load: (force: boolean, signal: AbortSignal) => Promise<T>) {
    this.#load = load;
  }

  async run(force = false): Promise<void> {
    this.#controller?.abort();
    const controller = new AbortController();
    this.#controller = controller;
    this.status = 'loading';
    this.error = '';
    try {
      const value = await this.#load(force, controller.signal);
      if (controller.signal.aborted) return;
      this.value = value;
      this.status = 'done';
    } catch (error) {
      if (controller.signal.aborted) return;
      this.error = error instanceof Error ? error.message : String(error);
      this.status = 'error';
      if (error instanceof ApiError && error.authProblem) toasts.error(error);
    }
  }

  cancel(): void {
    this.#controller?.abort();
    if (this.status === 'loading') this.status = this.value === undefined ? 'idle' : 'done';
  }
}
