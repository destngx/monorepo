import { StorageError, isAppError } from './errors';
import { calculateTtlUntilNext5am } from './ttl-utils';

const cache = new Map<string, { data: string; expiresAt: number }>();
const cacheTags = new Map<string, Set<string>>();

let upstashClient: UpstashCacheClient | null = null;

class UpstashCacheClient {
  private url: string;
  private token: string;

  constructor(url: string, token: string) {
    this.url = url.endsWith('/') ? url.slice(0, -1) : url;
    this.token = token;
  }

  async get(key: string): Promise<string | null> {
    try {
      const response = await fetch(`${this.url}/get/${key}`, {
        headers: { Authorization: `Bearer ${this.token}` },
      });
      if (!response.ok) return null;
      const result = await response.json();
      return result.result || null;
    } catch (error) {
      console.warn(`[Upstash] GET error for ${key}:`, error);
      return null;
    }
  }

  async set(key: string, value: string, exSeconds: number): Promise<boolean> {
    try {
      const response = await fetch(`${this.url}/pipeline`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify([['SET', key, value, 'EX', exSeconds.toString()]]),
      });
      return response.ok;
    } catch (error) {
      console.warn(`[Upstash] SET error for ${key}:`, error);
      return false;
    }
  }

  async delete(key: string): Promise<boolean> {
    try {
      const response = await fetch(`${this.url}/pipeline`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify([['DEL', key]]),
      });
      return response.ok;
    } catch (error) {
      console.warn(`[Upstash] DEL error for ${key}:`, error);
      return false;
    }
  }
}

function initUpstashClient(): void {
  const url = process.env.UPSTASH_REDIS_REST_URL;
  const token = process.env.UPSTASH_REDIS_REST_TOKEN;

  if (!url || !token) {
    console.warn('[Cache] Upstash Redis not configured, using in-memory cache');
    return;
  }

  upstashClient = new UpstashCacheClient(url, token);
  console.log('[Cache] Upstash Redis initialized');
}

export async function getCached<T>(key: string): Promise<T | null> {
  if (!upstashClient) {
    initUpstashClient();
  }

  if (upstashClient) {
    try {
      const cached = await upstashClient.get(key);
      if (cached) {
        return JSON.parse(cached) as T;
      }
    } catch (error) {
      console.warn(`[Cache] Upstash retrieval error for ${key}:`, error);
    }
  }

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

export async function setCache(key: string, data: any, ttlSeconds?: number) {
  if (!upstashClient) {
    initUpstashClient();
  }

  const ttl = ttlSeconds ?? calculateTtlUntilNext5am();

  const expiresAt = Date.now() + ttl * 1000;
  cache.set(key, {
    data: JSON.stringify(data),
    expiresAt,
  });

  if (upstashClient) {
    try {
      await upstashClient.set(key, JSON.stringify(data), ttl);
      console.debug(`[Cache] Stored: ${key} (TTL: ${ttl}s, expires at 5am GMT+7)`);
    } catch (error) {
      console.warn(`[Cache] Upstash set error for ${key}:`, error);
    }
  }
}

export async function invalidateCache(keyPrefix: string) {
  const keysToDelete: string[] = [];

  for (const key of cache.keys()) {
    if (key.startsWith(keyPrefix)) {
      keysToDelete.push(key);
      cache.delete(key);
    }
  }

  if (upstashClient && keysToDelete.length > 0) {
    for (const key of keysToDelete) {
      try {
        await upstashClient.delete(key);
      } catch (error) {
        console.warn(`[Cache] Upstash delete error for ${key}:`, error);
      }
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
  const tagSet = cacheTags.get(tag);
  if (tagSet) tagSet.add(key);
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

    if (upstashClient) {
      try {
        await upstashClient.delete(key);
      } catch (error) {
        console.warn(`[Cache] Upstash delete error for ${key}:`, error);
      }
    }
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

      if (upstashClient) {
        try {
          await upstashClient.delete(key);
        } catch (error) {
          console.warn(`[Cache] Upstash delete error for ${key}:`, error);
        }
      }
    }
  }

  console.log(`[Cache] Invalidated ${totalInvalidated} keys for ticker ${symbol} (reason: ${reason})`);
  return totalInvalidated;
}
export async function getCachedOrFetch<T>(
  key: string,
  fetchFn: () => Promise<T>,
  ttlSeconds?: number,
  forceFresh = false,
): Promise<T> {
  if (!forceFresh) {
    const cached = await getCached<T>(key);
    if (cached !== null) return cached;
  }

  const data = await fetchFn();
  const ttl = ttlSeconds ?? calculateTtlUntilNext5am();
  await setCache(key, data, ttl);
  return data;
}

export function withCache<T>(keyPrefix: string, handler: (...args: any[]) => Promise<T>, ttlSeconds?: number) {
  return async (...args: any[]): Promise<T> => {
    let forceFresh = false;
    let actualArgs = args;

    if (args.length > 0 && typeof args[args.length - 1] === 'boolean') {
      forceFresh = args[args.length - 1];
      actualArgs = args.slice(0, -1);
    }

    const key = `${keyPrefix}:${JSON.stringify(actualArgs)}`;
    const ttl = ttlSeconds ?? calculateTtlUntilNext5am();
    return getCachedOrFetch(key, () => handler(...args), ttl, forceFresh);
  };
}
