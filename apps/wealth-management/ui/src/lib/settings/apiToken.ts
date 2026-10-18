/** The engine's rule for API tokens (domain.NormalizeAPIToken). */
export const API_TOKEN_MIN_LENGTH = 32;
export const API_TOKEN_MAX_LENGTH = 256;

/** 32 random bytes as 64 hex characters, like `openssl rand -hex 32`. */
export function generateApiToken(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(32));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

/** Why the engine would refuse token (already trimmed), or '' when it is fine. */
export function apiTokenProblem(token: string): string {
  if (token.length < API_TOKEN_MIN_LENGTH) return `Use at least ${API_TOKEN_MIN_LENGTH} characters, or Generate one.`;
  if (token.length > API_TOKEN_MAX_LENGTH) return `Use at most ${API_TOKEN_MAX_LENGTH} characters.`;
  if (!/^[!-~]+$/.test(token)) return 'Use visible ASCII characters only, no spaces.';
  return '';
}
