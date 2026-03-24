'use client';

import { useState, useCallback, ReactNode } from 'react';
import { ErrorNotificationsContext } from '@wealth-management/hooks';
import type { ErrorNotification } from '@wealth-management/hooks';

export function ErrorNotificationsProvider({ children }: { children: ReactNode }) {
  const [notifications, setNotifications] = useState<ErrorNotification[]>([]);

  const addError = useCallback((message: string, severity: 'error' | 'warning' | 'info' = 'error') => {
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

    return () => clearTimeout(timer);
  }, []);

  const removeError = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

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
