import { AppError } from './AppError';
import { ErrorCode } from './types';

export function isAppError(error: unknown): error is AppError {
  return error instanceof AppError;
}

export function getErrorMessage(error: unknown): string {
  if (error instanceof AppError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

export function getUserMessage(error: unknown): string {
  if (error instanceof AppError) {
    return error.userMessage;
  }
  return 'An unexpected error occurred. Please try again later.';
}

export async function tryCatch<T>(
  fn: () => Promise<T>,
  onError?: (error: AppError) => AppError,
): Promise<[T | null, AppError | null]> {
  try {
    const result = await fn();
    return [result, null];
  } catch (error) {
    const appError = isAppError(error) ? error : new AppError(getErrorMessage(error));
    const handledError = onError ? onError(appError) : appError;
    return [null, handledError];
  }
}
