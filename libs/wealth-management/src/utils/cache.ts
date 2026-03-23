import { StorageError, isAppError } from './errors';

const cache = new Map<string, { data: string; expiresAt: number }>();

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
