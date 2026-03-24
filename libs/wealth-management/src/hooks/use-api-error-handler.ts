'use client';

import { useCallback } from 'react';
import { useErrorNotifications } from './use-error-notifications';
import { AppError } from '../utils/errors';
import { extractUserMessage } from '../utils/api-error-handler';

export function useApiErrorHandler() {
  const { addError } = useErrorNotifications();

  const handleError = useCallback(
    (error: unknown, fallbackMessage?: string) => {
      let message = fallbackMessage || 'An error occurred';

      if (error instanceof AppError) {
        message = error.userMessage;
      } else if (error instanceof Error) {
        message = error.message;
      }

      addError(message, 'error');
    },
    [addError],
  );

  const withErrorHandling = useCallback(
    async <T>(asyncFn: () => Promise<T>, fallbackMessage?: string): Promise<T | null> => {
      try {
        return await asyncFn();
      } catch (error) {
        handleError(error, fallbackMessage);
        return null;
      }
    },
    [handleError],
  );

  return {
    handleError,
    withErrorHandling,
    addError,
  };
}
