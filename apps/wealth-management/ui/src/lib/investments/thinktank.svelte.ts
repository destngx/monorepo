import { api } from '$lib/api/endpoints';
import { ApiError } from '$lib/api/client';
import type { NewsReport } from '$lib/api/types';
import { toasts } from '$lib/state/toasts.svelte';
import { fail, finish, initialState, reduce, startState, stop, type ThinkTankState } from './thinktank';

/**
 * The tab's one think tank run. It lives at module level so a run of several
 * minutes keeps going, and its result stays, when the user browses away.
 */
class ThinkTankRun {
  state = $state.raw<ThinkTankState>(initialState());
  #controller: AbortController | null = null;

  get running(): boolean {
    return this.state.status === 'running';
  }

  async start(news: NewsReport[] = []): Promise<void> {
    this.#controller?.abort();
    const controller = new AbortController();
    this.#controller = controller;
    this.state = startState(news);
    try {
      for await (const event of api.investmentAnalysis(news, controller.signal)) {
        if (controller.signal.aborted) break;
        this.state = reduce(this.state, event);
      }
      if (!controller.signal.aborted) this.state = finish(this.state);
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && error.authProblem) toasts.error(error);
      this.state = fail(this.state, error instanceof Error ? error.message : String(error));
    } finally {
      if (this.#controller === controller) this.#controller = null;
    }
  }

  stop(): void {
    if (!this.#controller) return;
    this.#controller.abort();
    this.#controller = null;
    this.state = stop(this.state);
  }

  reset(): void {
    this.stop();
    this.state = initialState();
  }
}

export const thinkTank = new ThinkTankRun();
