import type { Settled } from '$lib/api/load';
import type { EngineHealth } from '$lib/api/types';

/**
 * Engine health (WM-OPS-6) as loaded in the background by the root layout:
 * pages render before it arrives, and the dependency banner, the sync
 * control and the AI entry points appear once it does.
 */
class EngineHealthState {
  /** Null while the first check is in flight; kept while a re-check runs. */
  settled = $state<Settled<EngineHealth> | null>(null);

  /** The health report, or null when the engine has not answered (yet). */
  get health(): EngineHealth | null {
    return this.settled?.ok ? this.settled.value : null;
  }

  get checked(): boolean {
    return this.settled !== null;
  }

  /** The engine did not answer the health check. */
  get unreachable(): boolean {
    return this.settled !== null && !this.settled.ok;
  }

  ready(name: string): boolean {
    return this.health?.checks.some((check) => check.name === name && check.status === 'ok') ?? false;
  }
}

export const engineHealth = new EngineHealthState();
