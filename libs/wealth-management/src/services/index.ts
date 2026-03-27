// Only re-export generic API service clients that are client-safe (if any)
// Most sheet functions are server-side only.

export * from './services/exchange-rate-service';
export type { MarketAsset } from './services/market-pulse/types';
// export * from './services/investment-service';  // Server-only, don't export
// export * from './services/market-data-service';  // Imports search-service which imports fs
// export * from './services/news-service';  // Imports search-service which imports fs
// export * from './services/price-service';  // Imports search-service which imports fs
// export * from './services/search-service';  // Imports fs/node modules, server-only
export * from './services/stock-analysis-service';
export { fmarketApi } from './fmarket-api';

// sheet functions (exported in server.ts but some shared types/functions can go here if needed)
// For now, keep it simple and ensure the app uses /server for these.
