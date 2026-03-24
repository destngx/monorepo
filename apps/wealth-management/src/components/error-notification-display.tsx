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
    <div className="fixed top-6 right-6 z-[100] flex flex-col gap-3 max-w-sm w-full">
      {notifications.map((notification) => (
        <div
          key={notification.id}
          className={`group pointer-events-auto flex items-start gap-4 p-4 rounded-xl shadow-[0_8px_32px_rgba(0,0,0,0.12)] border backdrop-blur-md animate-in fade-in slide-in-from-right-8 duration-300 transition-all ${getSeverityStyles(notification.severity)}`}
        >
          <div className="flex-shrink-0 mt-0.5">
            {getIcon(notification.severity)}
          </div>
          <div className="flex-1 min-w-0 pr-2">
            <p className="text-[0.925rem] font-medium leading-relaxed tracking-tight break-words">
              {notification.message}
            </p>
          </div>
          {notification.dismissible && (
            <button
              onClick={() => removeError(notification.id)}
              className="flex-shrink-0 opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity p-1 rounded-md hover:bg-black/5 dark:hover:bg-white/5"
              aria-label="Dismiss notification"
            >
              <X size={14} className="text-muted-foreground" />
            </button>
          )}
        </div>
      ))}
    </div>
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
