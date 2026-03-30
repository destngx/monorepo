'use client';

import { createContext, useContext } from 'react';

export interface ErrorNotification {
  id: string;
  message: string;
  severity: 'error' | 'warning' | 'info';
  timestamp: number;
  dismissible?: boolean;
}

interface ErrorNotificationsContextType {
  notifications: ErrorNotification[];
  addError: (message: string, severity?: 'error' | 'warning' | 'info') => void;
  removeError: (id: string) => void;
  clearAll: () => void;
}

export const ErrorNotificationsContext = createContext<ErrorNotificationsContextType | undefined>(undefined);

export function useErrorNotifications(): ErrorNotificationsContextType {
  const context = useContext(ErrorNotificationsContext);
  if (!context) {
    throw new Error('useErrorNotifications must be used within ErrorNotificationsProvider');
  }
  return context;
}
