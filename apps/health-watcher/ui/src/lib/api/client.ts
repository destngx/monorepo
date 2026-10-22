import { postForm } from '../xhr';

/** Machine-readable error codes of the engine's error envelope. */
export type ErrorCode =
  'VALIDATION' | 'UNAUTHORIZED' | 'FORBIDDEN' | 'NOT_FOUND' | 'CONFLICT' | 'RATE_LIMITED' | 'TOO_LARGE' | 'INTERNAL';

/** An engine response with a non-2xx status, carrying the server's message. */
export class ApiError extends Error {
  readonly status: number;
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
}

export type Query = Record<string, string | number | boolean | undefined | null>;

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  query?: Query;
  body?: unknown;
  /** SvelteKit's load fetch, so requests are tracked by invalidation. */
  fetch?: typeof fetch;
  signal?: AbortSignal;
}

/** The session's CSRF token, sent on every write (HW-AUTH-2). Set by the session state. */
let csrfToken = '';

export function setCSRFToken(token: string): void {
  csrfToken = token;
}

/** Called on 401 from any request except the auth probes; the session state sends the user to login. */
let onUnauthorized: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler;
}

export function buildPath(path: string, query?: Query): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null && value !== '') params.set(key, String(value));
  }
  const search = params.toString();
  return `/api${path}${search ? `?${search}` : ''}`;
}

function headersFor(options: RequestOptions): Headers {
  const headers = new Headers();
  // FormData sets its own multipart boundary.
  if (options.body !== undefined && !(options.body instanceof FormData))
    headers.set('Content-Type', 'application/json');
  if ((options.method ?? 'GET') !== 'GET' && csrfToken) headers.set('X-CSRF-Token', csrfToken);
  return headers;
}

async function errorDetails(response: Response): Promise<{ message: string; code?: ErrorCode }> {
  const text = await response.text();
  try {
    const payload = JSON.parse(text) as { error?: unknown; code?: unknown };
    const code = typeof payload.code === 'string' && payload.code ? (payload.code as ErrorCode) : undefined;
    if (typeof payload.error === 'string' && payload.error) return { message: payload.error, code };
  } catch {
    // Not JSON: fall through to the raw body.
  }
  if (response.status === 502 || response.status === 504) {
    return { message: 'The engine is not reachable. Is it running?' };
  }
  return { message: text.trim() || `${response.status} ${response.statusText}` };
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const doFetch = options.fetch ?? fetch;
  const response = await doFetch(buildPath(path, options.query), {
    method: options.method ?? 'GET',
    headers: headersFor(options),
    body: options.body === undefined || options.body instanceof FormData ? options.body : JSON.stringify(options.body),
    credentials: 'same-origin',
    signal: options.signal,
  });
  if (!response.ok) {
    const { message, code } = await errorDetails(response);
    if (response.status === 401 && !path.startsWith('/auth/')) onUnauthorized?.();
    throw new ApiError(response.status, message, code, path);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

/** POSTs a form with upload progress, with the session's cookie and CSRF token. */
export async function uploadForm<T>(path: string, form: FormData, onProgress?: (fraction: number) => void): Promise<T> {
  const res = await postForm(buildPath(path), form, {
    headers: csrfToken ? { 'X-CSRF-Token': csrfToken } : {},
    withCredentials: true,
    onProgress,
  });
  if (res.status < 200 || res.status >= 300) {
    if (res.status === 401) onUnauthorized?.();
    const message =
      typeof res.body.error === 'string' && res.body.error ? res.body.error : `Upload failed (${res.status})`;
    const code = typeof res.body.code === 'string' ? (res.body.code as ErrorCode) : undefined;
    throw new ApiError(res.status, message, code, path);
  }
  return res.body as T;
}

/** The URL of a stored file, for <img> and <iframe>; the cookie authorises it. */
export function fileURL(tenantId: string, fileId: string, download = false): string {
  return buildPath(`/tenants/${tenantId}/files/${fileId}`, { download: download ? 1 : undefined });
}

/** The URL of a page's rendered image, or the masked copy the AI provider saw. */
export function pageImageURL(tenantId: string, pageId: string, masked = false): string {
  return buildPath(`/tenants/${tenantId}/pages/${pageId}/image`, { masked: masked ? 1 : undefined });
}
