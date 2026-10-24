/** An engine response with a non-2xx status, carrying the server's message and code. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string | undefined;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

/** The path of one agent's API, with the pane id encoded (it holds a colon). */
export function agentPath(paneId: string, action: 'screen' | 'prompt' | 'keys'): string {
  return `/api/agents/${encodeURIComponent(paneId)}/${action}`;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, { credentials: 'same-origin', ...init });
  } catch {
    throw new ApiError(0, 'The laptop is not reachable. Is Tailscale on?', 'UNREACHABLE');
  }
  if (!response.ok) {
    const text = await response.text();
    let message = text.trim() || `${response.status} ${response.statusText}`;
    let code: string | undefined;
    try {
      const body = JSON.parse(text) as { error?: unknown; code?: unknown };
      if (typeof body.error === 'string' && body.error) message = body.error;
      if (typeof body.code === 'string' && body.code) code = body.code;
    } catch {
      // Not JSON (a proxy's error page): keep the raw text.
    }
    if (response.status === 502 && !code) message = 'The engine is not running on the laptop.';
    throw new ApiError(response.status, message, code);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

function post(path: string, body: unknown): Promise<void> {
  return request<void>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export const api = {
  screen: (paneId: string, signal?: AbortSignal) =>
    request<{ text: string }>(agentPath(paneId, 'screen'), { signal, cache: 'no-store' }),
  prompt: (paneId: string, text: string) => post(agentPath(paneId, 'prompt'), { text }),
  keys: (paneId: string, keys: string[]) => post(agentPath(paneId, 'keys'), { keys }),
};
