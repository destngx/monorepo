import { api } from '$lib/api/endpoints';

// Quick capture drafts waiting for review (WM-CAP-19), shown as a nav badge.
class PendingDrafts {
  /** Null when unknown: the engine has no capture inbox or did not answer. */
  count = $state<number | null>(null);

  async refresh(): Promise<void> {
    try {
      this.count = (await api.captureInbox()).pendingDrafts;
    } catch {
      // Only the badge depends on it; the drafts page reports the error.
      this.count = null;
    }
  }
}

export const pendingDrafts = new PendingDrafts();
