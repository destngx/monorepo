import { apiToken } from '$lib/state/settings.svelte';

/** Machine-readable error codes of the engine's error envelope (WM-UI-4). */
export type ErrorCode =
  | 'VALIDATION'
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'SHEETS_NOT_CONFIGURED'
  | 'OAUTH_EXPIRED'
  | 'UPSTREAM'
  | 'UNAVAILABLE'
  | 'INTERNAL';

/**
 * A Google Sheets credential problem: `missing` needs the one-time setup,
 * `expired` needs a fresh refresh token.
 */
export type SheetsAuthKind = 'missing' | 'expired';

/** An engine response with a non-2xx status, carrying the server's message. */
export class ApiError extends Error {
  readonly status: number;
  /** From the error envelope; absent on responses from older engines. */
  readonly code: ErrorCode | undefined;
  /** The API path that failed, without the /api prefix or query. */
  readonly path: string;

  constructor(status: number, message: string, code?: ErrorCode, path = '') {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.path = path;
  }

  /** What to do when an optional service (AI, market data) is not configured or down; null otherwise. */
  get unavailableHint(): string | null {
    return this.code === 'UNAVAILABLE' ? unavailableHint(this.path) : null;
  }

  /** Which Google Sheets credential problem this is, if any (WM-UI-4). */
  get sheetsAuth(): SheetsAuthKind | null {
    return sheetsAuthKind(this.code, this.message);
  }

  /** Google Sheets credentials are missing or expired (WM-UI-4). */
  get authProblem(): boolean {
    return this.sheetsAuth !== null;
  }
}

const sheetsAuthPattern = /invalid_grant|oauth2|token (has been )?expired|missing google sheets configuration/i;
const sheetsMissingPattern = /missing google sheets configuration|not configured/i;

/**
 * UNAVAILABLE is an optional service (AI or a market provider) that is not
 * configured or is down; it never means Google Sheets, which has its own code.
 */
export function unavailableHint(path: string): string {
  return path.startsWith('/ai')
    ? 'AI is not configured or not reachable. Check Settings > AI.'
    : 'This service is not configured or not reachable right now.';
}

export function isSheetsAuthProblem(message: string): boolean {
  return sheetsAuthPattern.test(message);
}

/**
 * Classifies a failure by its envelope code; responses without a code (older
 * engines) fall back to matching the message.
 */
export function sheetsAuthKind(code: string | undefined, message: string): SheetsAuthKind | null {
  if (code === 'SHEETS_NOT_CONFIGURED') return 'missing';
  if (code === 'OAUTH_EXPIRED') return 'expired';
  if (code) return null;
  if (!isSheetsAuthProblem(message)) return null;
  return sheetsMissingPattern.test(message) ? 'missing' : 'expired';
}

export type Query = Record<string, string | number | boolean | undefined | null>;

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  query?: Query;
  body?: unknown;
  headers?: Record<string, string>;
  /** SvelteKit's load fetch, so requests are tracked by invalidation. */
  fetch?: typeof fetch;
  signal?: AbortSignal;
}

export function buildPath(path: string, query?: Query): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null && value !== '') params.set(key, String(value));
  }
  const search = params.toString();
  return `/api${path}${search ? `?${search}` : ''}`;
}

// Writes need the bearer token (WM-SEC-1); the engine ignores it on reads.
function headersFor(options: RequestOptions): Headers {
  const headers = new Headers(options.headers);
  // FormData sets its own multipart boundary.
  if (options.body !== undefined && !(options.body instanceof FormData))
    headers.set('Content-Type', 'application/json');
  const token = apiToken.value;
  if (token && (options.method ?? 'GET') !== 'GET') headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

async function errorDetails(response: Response): Promise<{ message: string; code?: ErrorCode }> {
  const text = await response.text();
  let code: ErrorCode | undefined;
  try {
    const payload = JSON.parse(text) as { error?: unknown; message?: unknown; code?: unknown };
    if (typeof payload.code === 'string' && payload.code) code = payload.code as ErrorCode;
    if (typeof payload.error === 'string' && payload.error) return { message: payload.error, code };
    if (typeof payload.message === 'string' && payload.message) return { message: payload.message, code };
  } catch {
    // Not JSON: fall through to the raw body.
  }
  if (response.status === 401) {
    return { message: 'API token missing or wrong. Set it in Settings to make changes.', code };
  }
  if (response.status === 403) {
    return {
      message: 'Writes are disabled: no API token is set. Open the engine on localhost and set one in Settings.',
      code,
    };
  }
  return { message: text.trim() || `${response.status} ${response.statusText}`, code };
}

async function send(path: string, options: RequestOptions): Promise<Response> {
  const doFetch = options.fetch ?? fetch;
  const response = await doFetch(buildPath(path, options.query), {
    method: options.method ?? 'GET',
    headers: headersFor(options),
    body: options.body === undefined || options.body instanceof FormData ? options.body : JSON.stringify(options.body),
    signal: options.signal,
  });
  if (!response.ok) {
    const { message, code } = await errorDetails(response);
    throw new ApiError(response.status, message, code, path);
  }
  return response;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await send(path, options);
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

/** Like request, but returns the response body stream (NDJSON endpoints). */
export async function requestStream(path: string, options: RequestOptions = {}): Promise<ReadableStream<Uint8Array>> {
  const response = await send(path, options);
  if (!response.body) throw new ApiError(response.status, 'The server sent no response body.');
  return response.body;
}
