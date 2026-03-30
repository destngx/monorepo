'use client';

import { useEffect } from 'react';
import { toast } from '@wealth-management/utils';

interface ServerErrorNotifierProps {
  errors?: string[];
}

/**
 * A utility component that triggers client-side toasts for errors
 * that occurred during server-side rendering/fetching.
 */
export function ServerErrorNotifier({ errors }: ServerErrorNotifierProps) {
  useEffect(() => {
    if (errors && errors.length > 0) {
      errors.forEach((err) => {
        // Use a small delay for each to avoid overlapping animations if multiple errors
        const timer = setTimeout(() => {
          toast.error(err);
        }, 100);
        return () => clearTimeout(timer);
      });
    }
  }, [errors]);

  return null;
}
