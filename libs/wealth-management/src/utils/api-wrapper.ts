import { AppError } from '../errors';
import { handleApiError } from './api-error-handler';

export async function fetchWithErrorHandling<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options);

  if (!response.ok) {
    const error = await handleApiError(response);
    throw error;
  }

  return response.json();
}

export function withErrorNotification<T extends (...args: any[]) => Promise<any>>(fn: T): T {
  return (async (...args: any[]) => {
    try {
      return await fn(...args);
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw new AppError(error instanceof Error ? error.message : 'An error occurred', undefined, 500, undefined, {
        context: { originalError: error },
        userMessage: 'An unexpected error occurred. Please try again.',
      });
    }
  }) as T;
}
