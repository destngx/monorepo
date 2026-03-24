'use client';

import { useErrorNotifications } from '@wealth-management/hooks';
import { AlertCircle, AlertTriangle, Info, X } from 'lucide-react';
import { useEffect, useState } from 'react';

export function ErrorNotificationDisplay() {
  const { notifications, removeError } = useErrorNotifications();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) return null;

  return (
    <>
      <div className="fixed bottom-4 right-4 z-40 text-xs text-gray-500 pointer-events-none">
        Notifications: {notifications.length}
      </div>

      <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-md">
        {notifications.map((notification) => (
          <div
            key={notification.id}
            className={`flex items-start gap-3 p-4 rounded-lg shadow-lg border animate-in fade-in slide-in-from-top-2 duration-300 ${getSeverityStyles(notification.severity)}`}
          >
            {getIcon(notification.severity)}
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium break-words">{notification.message}</p>
            </div>
            {notification.dismissible && (
              <button
                onClick={() => removeError(notification.id)}
                className="flex-shrink-0 mt-0.5 ml-2 hover:opacity-70 transition-opacity"
                aria-label="Dismiss notification"
              >
                <X size={16} />
              </button>
            )}
          </div>
        ))}
      </div>
    </>
  );
}

function getSeverityStyles(severity: 'error' | 'warning' | 'info'): string {
  switch (severity) {
    case 'error':
      return 'bg-red-50 border-red-200 text-red-900 dark:bg-red-900/20 dark:border-red-800 dark:text-red-100';
    case 'warning':
      return 'bg-yellow-50 border-yellow-200 text-yellow-900 dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-100';
    case 'info':
      return 'bg-blue-50 border-blue-200 text-blue-900 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-100';
  }
}

function getIcon(severity: 'error' | 'warning' | 'info') {
  const iconProps = { size: 20, className: 'flex-shrink-0 mt-0.5' };
  switch (severity) {
    case 'error':
      return <AlertCircle {...iconProps} className={`${iconProps.className} text-red-600 dark:text-red-400`} />;
    case 'warning':
      return <AlertTriangle {...iconProps} className={`${iconProps.className} text-yellow-600 dark:text-yellow-400`} />;
    case 'info':
      return <Info {...iconProps} className={`${iconProps.className} text-blue-600 dark:text-blue-400`} />;
  }
}
