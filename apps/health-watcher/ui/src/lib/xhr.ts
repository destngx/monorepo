/**
 * POST a form with upload progress. fetch reports no upload progress, so
 * the phone's send-ahead queue (HW-IMP-11) uses XMLHttpRequest.
 */
export interface FormResponse {
  status: number;
  /** The parsed JSON body, or {} when there is none. */
  body: Record<string, unknown>;
}

export interface PostFormOptions {
  headers?: Record<string, string>;
  /** Send cookies: true for the signed-in app, false for a capture link. */
  withCredentials: boolean;
  onProgress?: (fraction: number) => void;
  /** Tests pass a fake; the browser's XMLHttpRequest otherwise. */
  xhr?: () => XMLHttpRequest;
}

export function postForm(url: string, form: FormData, options: PostFormOptions): Promise<FormResponse> {
  return new Promise((resolve, reject) => {
    const x = options.xhr?.() ?? new XMLHttpRequest();
    x.open('POST', url);
    x.withCredentials = options.withCredentials;
    for (const [k, v] of Object.entries(options.headers ?? {})) x.setRequestHeader(k, v);
    x.upload.onprogress = (e) => {
      if (e.lengthComputable && e.total > 0) options.onProgress?.(e.loaded / e.total);
    };
    x.onload = () => {
      let body: Record<string, unknown> = {};
      try {
        body = JSON.parse(x.responseText || '{}') as Record<string, unknown>;
      } catch {
        body = {};
      }
      resolve({ status: x.status, body });
    };
    x.onerror = () => reject(new Error('The Mac did not answer'));
    x.onabort = () => reject(new Error('The upload was stopped'));
    x.send(form);
  });
}
