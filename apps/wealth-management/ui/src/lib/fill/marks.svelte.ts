import type { FillResult, FillValues } from '$lib/api/types';
import { filledValues, sameValue } from './fill';

/**
 * Which fields AI filled (WM-FILL-6). A field stays marked until the user
 * edits it, so the mark compares the filled value with what the form holds.
 */
export class FillMarks {
  #values = $state<FillValues>({});

  /** Marks the fields a fill set. */
  take(result: FillResult): void {
    this.#values = filledValues(result);
  }

  clear(): void {
    this.#values = {};
  }

  /** Whether the field holds the value AI filled. */
  has(field: string, current: unknown): boolean {
    return field in this.#values && sameValue(this.#values[field], current);
  }

  get count(): number {
    return Object.keys(this.#values).length;
  }
}
