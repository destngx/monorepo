import type { Agent } from '$lib/agents';

export type Connection = 'connecting' | 'live' | 'offline';

interface Snapshot {
  agents: Agent[];
  error?: string;
  code?: string;
}

/**
 * The agent list, kept current by the engine's event stream. The stream is
 * open only while the page is visible: the engine stops asking herdr when
 * nobody watches, and iOS would freeze a background stream anyway.
 */
class Live {
  agents = $state<Agent[]>([]);
  /** Why herdr could not list agents, from the engine; empty when it could. */
  error = $state('');
  connection = $state<Connection>('connecting');
  /** Whether the first snapshot has arrived. */
  loaded = $state(false);
  /** When the last snapshot arrived. */
  updatedAt = $state<Date | null>(null);

  #source: EventSource | null = null;
  #users = 0;

  /** Starts watching; returns the function that stops. Call it from an effect. */
  watch(): () => void {
    this.#users++;
    if (this.#users === 1) {
      document.addEventListener('visibilitychange', this.#onVisibility);
      if (document.visibilityState === 'visible') this.#open();
    }
    return () => {
      this.#users--;
      if (this.#users > 0) return;
      document.removeEventListener('visibilitychange', this.#onVisibility);
      this.#close();
    };
  }

  find(paneId: string): Agent | undefined {
    return this.agents.find((a) => a.paneId === paneId);
  }

  #onVisibility = () => {
    if (document.visibilityState === 'visible') this.#open();
    else this.#close();
  };

  #open() {
    if (this.#source) return;
    this.connection = 'connecting';
    const source = new EventSource('/api/events');
    source.onopen = () => (this.connection = 'live');
    source.onmessage = (event: MessageEvent<string>) => {
      const snapshot = JSON.parse(event.data) as Snapshot;
      this.agents = snapshot.agents;
      this.error = snapshot.error ?? '';
      this.loaded = true;
      this.updatedAt = new Date();
      this.connection = 'live';
    };
    // EventSource retries by itself; a closed one (the engine refused it) is retried here.
    source.onerror = () => {
      this.connection = 'offline';
      if (source.readyState === EventSource.CLOSED) {
        this.#source = null;
        setTimeout(() => {
          if (this.#users > 0 && document.visibilityState === 'visible') this.#open();
        }, 3000);
      }
    };
    this.#source = source;
  }

  #close() {
    this.#source?.close();
    this.#source = null;
  }
}

export const live = new Live();
