import { apiToken } from '$lib/state/settings.svelte';

/** An engine response with a non-2xx status, carrying the server's message. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }

  /** Google Sheets credentials are missing or expired (WM-UI-4). */
  get authProblem(): boolean {
    return isSheetsAuthProblem(this.message);
  }
}

const sheetsAuthPattern = /invalid_grant|oauth2|token (has been )?expired|missing google sheets configuration/i;

export function isSheetsAuthProblem(message: string): boolean {
  return sheetsAuthPattern.test(message);
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
  if (options.body !== undefined) headers.set('Content-Type', 'application/json');
  const token = apiToken.value;
  if (token && (options.method ?? 'GET') !== 'GET') headers.set('Authorization', `Bearer ${token}`);
  return headers;
}

async function errorMessage(response: Response): Promise<string> {
  const text = await response.text();
  try {
    const payload = JSON.parse(text) as { error?: unknown; message?: unknown };
    if (typeof payload.error === 'string' && payload.error) return payload.error;
    if (typeof payload.message === 'string' && payload.message) return payload.message;
  } catch {
    // Not JSON: fall through to the raw body.
  }
  if (response.status === 401) return 'API token missing or wrong. Set it in Settings to make changes.';
  if (response.status === 403) return 'Writes are disabled: the server has no API_TOKEN and is not bound to localhost.';
  return text.trim() || `${response.status} ${response.statusText}`;
}

async function send(path: string, options: RequestOptions): Promise<Response> {
  const doFetch = options.fetch ?? fetch;
  const response = await doFetch(buildPath(path, options.query), {
    method: options.method ?? 'GET',
    headers: headersFor(options),
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  });
  if (!response.ok) {
    throw new ApiError(response.status, await errorMessage(response));
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
