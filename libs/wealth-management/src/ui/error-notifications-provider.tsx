'use client';

import { useState, useCallback, ReactNode, useEffect } from 'react';
import { ErrorNotificationsContext } from '@wealth-management/hooks';
import type { ErrorNotification } from '@wealth-management/hooks';
import { toast } from '@wealth-management/utils';

export function ErrorNotificationsProvider({ children }: { children: ReactNode }) {
  const [notifications, setNotifications] = useState<ErrorNotification[]>([]);

  const removeError = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const addError = useCallback(
    (message: string, severity: 'error' | 'warning' | 'info' = 'error') => {
      const id = `error-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const notification: ErrorNotification = {
        id,
        message,
        severity,
        timestamp: Date.now(),
        dismissible: true,
      };

      setNotifications((prev) => [...prev, notification]);

      const timer = setTimeout(() => {
        removeError(id);
      }, 5000);
    },
    [removeError],
  );

  useEffect(() => {
    // 1. Subscribe to global toast emitter
    const unsubscribeToast = toast.subscribe(({ message, severity }) => {
      addError(message, severity);
    });

    // 2. Global window errors
    const handleGlobalError = (event: ErrorEvent) => {
      addError(event.message || 'A terminal error occurred', 'error');
    };

    // 3. Unhandled promise rejections (like uncaught fetches)
    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      const message =
        event.reason instanceof Error
          ? event.reason.message
          : typeof event.reason === 'string'
            ? event.reason
            : 'An asynchronous error occurred';

      addError(message, 'error');
    };

    window.addEventListener('error', handleGlobalError);
    window.addEventListener('unhandledrejection', handleUnhandledRejection);

    // 4. Client-side fetch interceptor
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      try {
        const response = await originalFetch(...args);
        if (!response.ok) {
          // Attempt to extract error message from response body
          let userMessage = `Request failed: ${response.status} ${response.statusText}`;
          try {
            const data = await response.clone().json();
            if (data.userMessage) userMessage = data.userMessage;
            else if (data.message) userMessage = data.message;
            else if (data.error) userMessage = data.error;

            const isInvalidGrant = 
              data.code === 'invalid_grant' || 
              (data.message && data.message.includes('invalid_grant')) ||
              (data.error && data.error.includes('invalid_grant'));

            if (isInvalidGrant) {
              userMessage = 'Google access session has expired or permissions were revoked. Please sign in again or re-connect your accounts.';
            }
          } catch (e) {
            // Ignore if not JSON
          }
          addError(userMessage, 'error');
        }
        return response;
      } catch (error) {
        addError(error instanceof Error ? error.message : 'Network failure', 'error');
        throw error;
      }
    };

    return () => {
      window.fetch = originalFetch;
      unsubscribeToast();
      window.removeEventListener('error', handleGlobalError);
      window.removeEventListener('unhandledrejection', handleUnhandledRejection);
    };
  }, [addError]);

  const clearAll = useCallback(() => {
    setNotifications([]);
  }, []);

  return (
    <ErrorNotificationsContext.Provider
      value={{
        notifications,
        addError,
        removeError,
        clearAll,
      }}
    >
      {children}
    </ErrorNotificationsContext.Provider>
  );
}
