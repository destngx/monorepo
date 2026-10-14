import { api } from '$lib/api/endpoints';
import type { AIProviderUsage } from '$lib/api/types';
import { onAIRequestDone } from './activity';

/** How often the menu bar gauge refreshes the quota while the tab is visible. */
export const AI_USAGE_REFRESH_MS = 5 * 60 * 1000;
/** Quiet time after the last AI request before the quota is refetched, so a burst refetches once. */
export const AI_USAGE_SETTLE_MS = 1000;

// OpenAI quota shared by the menu bar gauge and Settings (WM-SET-4), so a
// refresh in one place updates both.
class AIUsageState {
  usage = $state<AIProviderUsage | null>(null);
  error = $state('');
  loading = $state(false);
  loadedAt = $state(new Date());
  #pending: Promise<void> | null = null;
  #settle: ReturnType<typeof setTimeout> | undefined;

  /** Concurrent callers share one request. */
  load(): Promise<void> {
    this.#pending ??= this.#fetch().finally(() => (this.#pending = null));
    return this.#pending;
  }

  /** Refetches once AI requests have stopped finishing for a moment. */
  refreshSoon() {
    clearTimeout(this.#settle);
    // A fetch already in flight may predate the request: read again after it.
    this.#settle = setTimeout(
      () => void (this.#pending ?? Promise.resolve()).then(() => this.load()),
      AI_USAGE_SETTLE_MS,
    );
  }

  async #fetch() {
    this.loading = true;
    try {
      this.usage = await api.aiUsage();
      this.error = '';
      this.loadedAt = new Date();
    } catch (failure) {
      this.usage = null;
      this.error = failure instanceof Error ? failure.message : String(failure);
    } finally {
      this.loading = false;
    }
  }
}

export const aiUsage = new AIUsageState();

// Every AI request may spend quota: show what is left after it.
onAIRequestDone(() => aiUsage.refreshSoon());
