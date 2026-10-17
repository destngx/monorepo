import { ApiError } from '$lib/api/client';
import { api } from '$lib/api/endpoints';
import { defer, type Streamed } from '$lib/api/load';
import type { AIStatus } from '$lib/api/types';

type Fetch = typeof fetch;

/**
 * The engine's AI status for a research page that runs AI flows, streamed.
 * The AI routes are absent when AI is off: that 404 resolves to null (AI not
 * available) instead of a failure, so it is never mistaken for a missing
 * Google Sheet. Other failures (engine down) stay failures with a Retry.
 */
export function loadAIStatus(fetch?: Fetch): Streamed<AIStatus | null> {
  return defer(() =>
    api.aiStatus(fetch).catch((cause: unknown) => {
      if (cause instanceof ApiError && cause.status === 404) return null;
      throw cause;
    }),
  );
}

/**
 * Why the AI flow cannot run, or null when it can. `undefined` (not loaded
 * yet) is null too: the page never waits for the status before it is usable.
 */
export function aiUnavailable(status: AIStatus | null | undefined, prompts: readonly string[]): string | null {
  if (status === undefined) return null;
  if (status === null) return 'AI is not available on the engine.';
  if (!status.configured) return 'AI is not configured: add the AI endpoint and model.';
  const missing = prompts.filter((key) => status.missingPrompts.includes(key));
  if (missing.length === 1) return `The Prompts sheet has no "${missing[0]}" prompt.`;
  if (missing.length) return `The Prompts sheet is missing: ${missing.join(', ')}.`;
  return null;
}
