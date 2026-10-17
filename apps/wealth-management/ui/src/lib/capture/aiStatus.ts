import { ApiError } from '$lib/api/client';
import { api } from '$lib/api/endpoints';
import type { AIStatus } from '$lib/api/types';

/**
 * The engine's AI status for the capture pages, or null when the engine has
 * no AI routes (AI off: it answers 404 and the page explains instead of
 * failing). Any other failure is kept, so the page shows it with a Retry.
 */
export async function captureAIStatus(fetch?: typeof globalThis.fetch): Promise<AIStatus | null> {
  try {
    return await api.aiStatus(fetch);
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 404) return null;
    throw cause;
  }
}
