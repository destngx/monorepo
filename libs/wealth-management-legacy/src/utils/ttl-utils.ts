const DEFAULT_MAX_TTL_SECONDS = 86400;
const RESET_HOUR_GMT_PLUS_7 = 5;

export function calculateTtlUntilNext5am(maxTtlSeconds: number = DEFAULT_MAX_TTL_SECONDS): number {
  const now = new Date();
  const nowUTC = now.getTime();

  const offsetMinutes = 7 * 60;
  const gmt7Time = new Date(nowUTC + offsetMinutes * 60 * 1000);

  const resetTime = new Date(gmt7Time);
  resetTime.setHours(RESET_HOUR_GMT_PLUS_7, 0, 0, 0);

  if (gmt7Time >= resetTime) {
    resetTime.setDate(resetTime.getDate() + 1);
  }

  const ttlMs = resetTime.getTime() - gmt7Time.getTime();
  const ttlSeconds = Math.floor(ttlMs / 1000);
  const cappedTtl = Math.min(ttlSeconds, maxTtlSeconds);

  console.debug(
    `[TTL] Current (GMT+7): ${gmt7Time.toISOString()} | ` +
      `Next reset: ${resetTime.toISOString()} | ` +
      `TTL: ${cappedTtl}s (${(cappedTtl / 3600).toFixed(1)}h)`,
  );

  return cappedTtl;
}

export function getCacheTtl(dataType = 'default', maxTtl: number = DEFAULT_MAX_TTL_SECONDS): number {
  return calculateTtlUntilNext5am(maxTtl);
}
