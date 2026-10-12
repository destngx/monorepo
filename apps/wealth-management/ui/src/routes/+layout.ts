import { api } from '$lib/api/endpoints';
import type { EngineHealth } from '$lib/api/types';
import type { LayoutLoad } from './$types';

// Client-rendered SPA: the Go engine serves the static shell and the browser does the rest.
export const ssr = false;

export const load: LayoutLoad = async ({ fetch }) => {
  let health: EngineHealth | null = null;
  try {
    health = await api.health(fetch);
  } catch {
    // Unreachable engine: the layout shows a banner instead of failing every page.
  }
  return { health };
};
