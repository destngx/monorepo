import { api } from '$lib/api/endpoints';
import { toasts } from './toasts.svelte';

export const AUTO_SYNC_MS = 15 * 60 * 1000;

// Sync (WM-UI-3) invalidates the engine's sheet caches, then reloads page
// data. It runs on start-up and every 15 minutes; failures always surface.
class SyncState {
  syncing = $state(false);
  lastSynced = $state<Date | null>(null);
  nextSyncAt = $state<number | null>(null);
  lastError = $state<string | null>(null);

  async run(reload: () => Promise<void>): Promise<boolean> {
    if (this.syncing) return false;
    this.syncing = true;
    try {
      await api.sync();
      this.lastSynced = new Date();
      this.lastError = null;
      await reload();
      return true;
    } catch (error) {
      this.lastError = error instanceof Error ? error.message : String(error);
      toasts.error(error, 'Sync failed');
      return false;
    } finally {
      this.syncing = false;
      this.nextSyncAt = Date.now() + AUTO_SYNC_MS;
    }
  }
}

export const sync = new SyncState();
