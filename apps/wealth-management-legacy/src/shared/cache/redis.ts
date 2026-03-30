import { Redis } from '@upstash/redis';

const validateRedisEnv = () => {
  const url = process.env.UPSTASH_REDIS_REST_URL;
  const token = process.env.UPSTASH_REDIS_REST_TOKEN;

  if (!url || !token) {
    console.warn(
      '[Redis] Missing credentials. Caching disabled. Set UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN',
    );
    return null;
  }

  return { url, token };
};

const redisConfig = validateRedisEnv();
export const redis = redisConfig
  ? new Redis({
      url: redisConfig.url,
      token: redisConfig.token,
    })
  : null;

export const isRedisAvailable = (): boolean => {
  return redis !== null;
};

export type CacheOperationResult<T> = {
  success: boolean;
  data?: T;
  error?: string;
};

export async function getCacheValue<T>(key: string): Promise<CacheOperationResult<T>> {
  if (!redis) {
    return { success: false, error: 'Redis not available' };
  }

  try {
    const value = await redis.get<T>(key);
    if (value === null) {
      return { success: true, data: undefined };
    }
    return { success: true, data: value };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[Redis] Get failed for key "${key}":`, message);
    return { success: false, error: message };
  }
}

export async function setCacheValue<T>(key: string, value: T, ttlSeconds: number): Promise<CacheOperationResult<void>> {
  if (!redis) {
    return { success: false, error: 'Redis not available' };
  }

  try {
    if (ttlSeconds > 0) {
      await redis.setex(key, ttlSeconds, value);
    } else {
      await redis.set(key, value);
    }
    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[Redis] Set failed for key "${key}":`, message);
    return { success: false, error: message };
  }
}

export async function deleteCacheValue(key: string): Promise<CacheOperationResult<void>> {
  if (!redis) {
    return { success: false, error: 'Redis not available' };
  }

  try {
    await redis.del(key);
    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[Redis] Delete failed for key "${key}":`, message);
    return { success: false, error: message };
  }
}

export async function cacheKeyExists(key: string): Promise<CacheOperationResult<boolean>> {
  if (!redis) {
    return { success: false, error: 'Redis not available' };
  }

  try {
    const exists = await redis.exists(key);
    return { success: true, data: exists === 1 };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[Redis] Exists check failed for key "${key}":`, message);
    return { success: false, error: message };
  }
}

export async function invalidateCachePattern(pattern: string): Promise<CacheOperationResult<number>> {
  if (!redis) {
    return { success: false, error: 'Redis not available' };
  }

  try {
    const keys = await redis.keys(pattern);
    if (keys.length === 0) {
      return { success: true, data: 0 };
    }

    await redis.del(...keys);
    return { success: true, data: keys.length };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    console.error(`[Redis] Pattern invalidation failed for "${pattern}":`, message);
    return { success: false, error: message };
  }
}

export async function getOrSetCache<T>(
  key: string,
  fetchFn: () => Promise<T>,
  ttlSeconds: number,
  forceFresh = false,
): Promise<T> {
  if (!forceFresh) {
    const cached = await getCacheValue<T>(key);
    if (cached.success && cached.data !== undefined) {
      return cached.data;
    }
  }

  const data = await fetchFn();

  setCacheValue(key, data, ttlSeconds).catch((err) => {
    console.error(`[Redis] Failed to cache result for key "${key}":`, err);
  });

  return data;
}
