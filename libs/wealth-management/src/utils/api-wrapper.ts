import { AppError } from './errors';
import { handleApiError } from './api-error-handler';
import { toast } from './toast';

export async function fetchWithErrorHandling<T>(url: string, options?: RequestInit): Promise<T> {
  try {
    const response = await fetch(url, options);

    if (!response.ok) {
      const error = await handleApiError(response);
      toast.error(error.userMessage);
      throw error;
    }

    return response.json();
  } catch (error) {
    if (error instanceof AppError) throw error;

    const message = error instanceof Error ? error.message : 'An unexpected error occurred';
    toast.error(message);
    throw error;
  }
}

export function withErrorNotification<T extends (...args: any[]) => Promise<any>>(fn: T): T {
  return (async (...args: any[]) => {
    try {
      return await fn(...args);
    } catch (error) {
      const appError =
        error instanceof AppError
          ? error
          : new AppError(error instanceof Error ? error.message : 'An error occurred', undefined, 500, undefined, {
              context: { originalError: error },
              userMessage: 'An unexpected error occurred. Please try again.',
            });

      toast.error(appError.userMessage);
      throw appError;
    }
  }) as T;
}
