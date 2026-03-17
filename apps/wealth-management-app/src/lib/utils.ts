import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Re-export utility functions from the utils directory
export * from './utils/currency';
export * from './utils/date';
export * from './utils/validation';
// Re-export utility functions from the utils directory
export * from './utils/currency';
export * from './utils/date';
export * from './utils/validation';

