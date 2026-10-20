import type { Settled } from '$lib/api/load';
import type { EngineHealth } from '$lib/api/types';
import { toasts } from './toasts.svelte';

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

  /**
   * Stores a new health answer. The sheets check reads the spreadsheet with
   * the engine's current token, so a pass means Google accepts it again (a
   * re-auth followed by a restart) and the re-auth alert from an earlier
   * failed request is out of date.
   */
  report(settled: Settled<EngineHealth> | null): void {
    this.settled = settled;
    if (this.ready('sheets')) toasts.sheetsAuthProblem = null;
  }

  ready(name: string): boolean {
    return this.health?.checks.some((check) => check.name === name && check.status === 'ok') ?? false;
  }
}

export const engineHealth = new EngineHealthState();
