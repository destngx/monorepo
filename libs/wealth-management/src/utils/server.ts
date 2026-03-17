import { NextResponse } from 'next/server';
import { GoogleSheetsError } from '../services/sheets/auth';

/**
 * Standardized API Error Handler
 *
 * Handles specialized error cases (like Google Sheets auth) while providing
 * a robust default for unknown internal errors with full stack trace logging.
 */
export function handleApiError(error: unknown, context?: string) {
  const errorMessage = context ? `[API Error] ${context}:` : '[API Error]:';

  // Log the error with stack trace for debugging
  console.error(errorMessage, error);

  // Handle specialized Google Sheets errors
  if (error instanceof GoogleSheetsError) {
    if (error.code === 'MISSING_CREDENTIALS' || error.code === 'OAUTH_EXPIRED') {
      return NextResponse.json(
        {
          error: error.message,
          code: error.code,
          instruction: 'Please run `pnpm run auth:setup` in your terminal to configure Google Sheets.',
        },
        { status: 401 },
      );
    }
  }

  // Default fallback for all other errors
  const message = error instanceof Error ? error.message : 'An internal server error occurred';

  return NextResponse.json(
    {
      error: message,
      // In development, we could expose more, but let's keep it safe
    },
    { status: 500 },
  );
}
