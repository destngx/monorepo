import { StorageError, isAppError } from './errors';

const cache = new Map<string, { data: string; expiresAt: number }>();
const cacheTags = new Map<string, Set<string>>(); // tag -> set of cache keys

export async function getCached<T>(key: string): Promise<T | null> {
  const entry = cache.get(key);
  if (!entry) return null;

  if (Date.now() > entry.expiresAt) {
    cache.delete(key);
    return null;
  }

  try {
    return JSON.parse(entry.data) as T;
  } catch (e) {
    const storageError = isAppError(e)
      ? e
      : new StorageError(`Failed to parse cache for key ${key}`, {
          context: { error: e, key },
          userMessage: 'Cache read error.',
        });
    console.error(storageError.message, storageError);
    return null;
  }
}

export async function setCache(key: string, data: any, ttlSeconds = 300) {
  const expiresAt = Date.now() + ttlSeconds * 1000;
  cache.set(key, {
    data: JSON.stringify(data),
    expiresAt,
  });
}

export async function invalidateCache(keyPrefix: string) {
  for (const key of cache.keys()) {
    if (key.startsWith(keyPrefix)) {
      cache.delete(key);
    }
  }
}

/**
 * Register a cache key with a tag for batch invalidation.
 * Useful for invalidating related caches when events occur (e.g., earnings release).
 */
export async function tagCache(tag: string, key: string) {
  if (!cacheTags.has(tag)) {
    cacheTags.set(tag, new Set());
  }
  cacheTags.get(tag).add(key);
}

/**
 * Invalidate all cache keys associated with a specific tag.
 * Example: invalidateCacheByTag('earnings:AAPL') invalidates all AAPL analysis caches.
 */
export async function invalidateCacheByTag(tag: string) {
  const keysToInvalidate = cacheTags.get(tag);
  if (!keysToInvalidate) {
    console.warn(`[Cache] No keys found for tag: ${tag}`);
    return 0;
  }

  let count = 0;
  for (const key of keysToInvalidate) {
    cache.delete(key);
    count++;
  }

  cacheTags.delete(tag);
  console.log(`[Cache] Invalidated ${count} keys for tag: ${tag}`);
  return count;
}

/**
 * Invalidate all caches related to a ticker (e.g., on earnings release or stock split).
 */
export async function invalidateTickerCache(symbol: string, reason: string) {
  const invalidationPatterns = [
    `stock-analysis:${symbol}`,
    `vnstock:stock:${symbol}`,
    `vnstock:historical:${symbol}`,
    `market-pulse:asset:${symbol}`,
    `search:`, // Note: search results may be reusable, but we invalidate if needed
  ];

  let totalInvalidated = 0;
  for (const pattern of invalidationPatterns) {
    const keysToDelete: string[] = [];
    for (const key of cache.keys()) {
      if (key.includes(pattern)) {
        keysToDelete.push(key);
      }
    }

    for (const key of keysToDelete) {
      cache.delete(key);
      totalInvalidated++;
    }
  }

  console.log(`[Cache] Invalidated ${totalInvalidated} keys for ticker ${symbol} (reason: ${reason})`);
  return totalInvalidated;
}
export async function getCachedOrFetch<T>(
  key: string,
  fetchFn: () => Promise<T>,
  ttlSeconds = 300,
  forceFresh = false,
): Promise<T> {
  if (!forceFresh) {
    const cached = await getCached<T>(key);
    if (cached !== null) return cached;
  }

  const data = await fetchFn();
  await setCache(key, data, ttlSeconds);
  return data;
}

export function withCache<T>(keyPrefix: string, handler: (...args: any[]) => Promise<T>, ttlSeconds = 300) {
  return async (...args: any[]): Promise<T> => {
    // Standard pattern: last boolean argument is forceFresh
    let forceFresh = false;
    let actualArgs = args;

    if (args.length > 0 && typeof args[args.length - 1] === 'boolean') {
      forceFresh = args[args.length - 1];
      actualArgs = args.slice(0, -1);
    }

    const key = `${keyPrefix}:${JSON.stringify(actualArgs)}`;
    return getCachedOrFetch(key, () => handler(...args), ttlSeconds, forceFresh);
  };
}
