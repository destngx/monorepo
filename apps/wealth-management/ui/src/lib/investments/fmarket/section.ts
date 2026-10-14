import type { DataSource } from '$lib/api/types';
import { sourceOf } from '$lib/source';

// Sections load independently, so one failing fmarket action never blanks the page.

export type Section<T> = { ok: true; value: T } | { ok: false; error: string };

export async function settle<T>(load: () => Promise<T>): Promise<Section<T>> {
  try {
    return { ok: true, value: await load() };
  } catch (cause) {
    return { ok: false, error: cause instanceof Error ? cause.message : String(cause) };
  }
}

/** A parsed fmarket action payload with where it came from (WM-INV-15). */
export interface Sourced<T> {
  data: T;
  source?: DataSource;
}

export function sourced<T>(payload: unknown, parse: (payload: unknown) => T): Sourced<T> {
  return { data: parse(payload), source: sourceOf(payload) };
}
