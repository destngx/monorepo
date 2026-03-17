// This file is used for client-safe exports (types, small constants, or client-safe functions)
// Most services and sheets code is server-only and should NOT be exported here.

export { AI_MODELS, type AIModelId } from './types/chat';

// Re-export specific sub-barrels that are client-safe
export * from './schemas';
export * from './types';
export * from './utils';
export * from './hooks';
export * from './features';

// NOTE: ai, services, and ui should generally be imported from their subpaths
// to avoid bundling server-side code or large UI components unnecessarily.
// Most services and sheets code is server-only and should NOT be exported here.

export { AI_MODELS, type AIModelId } from './types/chat';

// Re-export specific sub-barrels that are client-safe
export * from './schemas';
export * from './types';
export * from './utils';

// NOTE: ai, services, and ui should generally be imported from their subpaths
// to avoid bundling server-side code or large UI components unnecessarily.
