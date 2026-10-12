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
      // Engines without the error envelope answer 404 when the sheet is not
      // configured; newer ones answer 503 SHEETS_NOT_CONFIGURED.
      const isLegacyMissing = cause.status === 404 && cause.code === undefined;
      const message = isLegacyMissing
        ? 'This data is unavailable: Google Sheets is not configured on the engine.'
        : cause.message;
      const sheetsAuth = isLegacyMissing ? 'missing' : (cause.sheetsAuth ?? undefined);
      error(isLegacyMissing ? 503 : cause.status, {
        message,
        status: cause.status,
        authProblem: sheetsAuth !== undefined,
        sheetsAuth,
      });
    }
    error(502, { message: cause instanceof Error ? `Engine unreachable: ${cause.message}` : 'Engine unreachable' });
  }
}
