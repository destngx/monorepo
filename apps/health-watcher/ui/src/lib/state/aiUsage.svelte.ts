import { engine } from '$lib/api/endpoints';
import type { AIUsage } from '$lib/api/types';

// The AI provider's quota for the menu bar gauge (HW-OPS-10).
class AIUsageState {
  /** False while AI reading is off, or before the first answer. */
  enabled = $state(false);
  usage = $state<AIUsage | null>(null);
  /** Why the quota could not be read; empty when it was. */
  error = $state('');
  loadedAt = $state(new Date(0));
  #pending: Promise<void> | null = null;

  /** Concurrent callers share one request. */
  load(): Promise<void> {
    this.#pending ??= this.#fetch().finally(() => (this.#pending = null));
    return this.#pending;
  }

  async #fetch() {
    try {
      const view = await engine.aiUsage();
      this.enabled = view.enabled;
      this.usage = view.usage ?? null;
      this.error = view.error ?? '';
    } catch (failure) {
      this.usage = null;
      this.error = failure instanceof Error ? failure.message : String(failure);
    } finally {
      this.loadedAt = new Date();
    }
  }
}

export const aiUsage = new AIUsageState();
