// Connectivity (WM-UI-6): the app has no offline queue, so losing the network
// blocks the UI until it returns, then the data is re-synced.

export type NetworkChange = 'lost' | 'restored' | null;

class NetworkState {
  online = $state(true);

  /** Records the browser's status and reports the change, if any. */
  update(online: boolean): NetworkChange {
    if (online === this.online) return null;
    this.online = online;
    return online ? 'restored' : 'lost';
  }
}

export const network = new NetworkState();
