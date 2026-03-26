// Client-safe re-exports for AI logic
export { AI_MODELS, type AIModelId } from '@wealth-management/types/chat';

// Types are client-safe
export * from './core/types';

// DO NOT export prompt builders here — they depend on server-side Google Sheets API
// Import from './server' instead if you need them server-side
