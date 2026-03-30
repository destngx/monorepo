import { NextResponse } from 'next/server';
import { GoogleSheetsError } from '../services/sheets/auth';
import { AppError, NetworkError, isAppError, ErrorCode } from './errors';

export function handleApiError(error: unknown, context?: string) {
  const errorMessage = context ? `[API Error] ${context}:` : '[API Error]:';

  console.error(errorMessage, error);

  if (error instanceof GoogleSheetsError) {
    if (error.sheetsErrorCode === 'MISSING_CREDENTIALS' || error.sheetsErrorCode === 'OAUTH_EXPIRED') {
      return NextResponse.json(
        {
          error: error.message,
          code: error.sheetsErrorCode,
          instruction: 'Please run `pnpm run auth:setup` in your terminal to configure Google Sheets.',
        },
        { status: 401 },
      );
    }
  }

  const appError = isAppError(error)
    ? error
    : new AppError(error instanceof Error ? error.message : 'An internal server error occurred');

  return NextResponse.json(
    {
      error: appError.message,
    },
    { status: appError.statusCode },
  );
}
