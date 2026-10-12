import { error } from '@sveltejs/kit';
import { ApiError } from './client';

/**
 * Runs page-load requests and turns API failures into SvelteKit errors, so
 * +error.svelte can show the server message (and the re-auth hint).
 */
export async function loadOrError<T>(load: () => Promise<T>): Promise<T> {
  try {
    return await load();
  } catch (cause) {
    if (cause instanceof ApiError) {
      // Sheet routes are not registered when the sheet is not configured.
      const message =
        cause.status === 404
          ? 'This data is unavailable: Google Sheets is not configured on the engine.'
          : cause.message;
      error(cause.status === 404 ? 503 : cause.status, {
        message,
        status: cause.status,
        authProblem: cause.authProblem,
      });
    }
    error(502, { message: cause instanceof Error ? `Engine unreachable: ${cause.message}` : 'Engine unreachable' });
  }
}
