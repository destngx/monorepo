import type { Metric, MetricCategory } from '$lib/api/types';
import { metricCategoryLabels } from '$lib/format';
import { session } from './session.svelte';

/** The metric catalog (HW-MET-1), loaded once per tenant and shared by the pages. */
class CatalogState {
  metrics = $state<Metric[]>([]);
  #tenant = '';
  #pending: Promise<void> | null = null;

  readonly byCode = $derived(new Map(this.metrics.map((m) => [m.code, m])));

  /** The catalog grouped by category, in the categories' display order. */
  readonly groups = $derived(
    (Object.keys(metricCategoryLabels) as MetricCategory[])
      .map((category) => ({
        category,
        label: metricCategoryLabels[category],
        metrics: this.metrics.filter((m) => m.category === category),
      }))
      .filter((g) => g.metrics.length > 0),
  );

  load(fetch?: typeof globalThis.fetch): Promise<void> {
    const tenant = session.tenantId ?? '';
    if (tenant === this.#tenant && this.#pending) return this.#pending;
    this.#tenant = tenant;
    this.#pending = session.api.metrics(fetch).then(
      (list) => {
        this.metrics = list;
      },
      (error: unknown) => {
        this.#pending = null;
        throw error;
      },
    );
    return this.#pending;
  }
}

export const catalog = new CatalogState();
