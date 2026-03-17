// Simple in-memory cache to replace SQLite-backed cache
// This is per-instance and suitable for local development/single-user setup.

const cache = new Map<string, { data: string; expiresAt: number }>();

export function getCached<T>(key: string): Promise<T | null> {
  const entry = cache.get(key);
  if (!entry) return Promise.resolve(null);

  if (Date.now() > entry.expiresAt) {
    cache.delete(key);
    return Promise.resolve(null);
  }

  try {
    return Promise.resolve(JSON.parse(entry.data) as T);
  } catch (e) {
    // Keep the console message short and actionable for debugging.
    console.error(`Failed to parse cache for key ${key}:`, e);
    return Promise.resolve(null);
  }
}

export function setCache(key: string, data: any, ttlSeconds = 300): Promise<void> {
  const expiresAt = Date.now() + ttlSeconds * 1000;
  cache.set(key, {
    data: JSON.stringify(data),
    expiresAt,
  });
  return Promise.resolve();
}

export function invalidateCache(keyPrefix: string): Promise<void> {
  for (const key of cache.keys()) {
    if (key.startsWith(keyPrefix)) {
      cache.delete(key);
    }
  }
  return Promise.resolve();
}
