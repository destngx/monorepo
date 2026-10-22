import { postForm } from './xhr';

/** The phone capture page's link handling and upload (HW-IMP-9). */

const TOKEN_KEY = 'hw.capture.token';

/**
 * The upload token of a capture link. It arrives in the fragment, is kept in
 * sessionStorage so a reload (iOS unloading the page during the camera)
 * still works, and is removed from the address bar.
 */
export function takeCaptureToken(
  hash: string,
  path: string,
  storage: Storage,
  replaceURL: (url: string) => void,
): string | null {
  const fromLink = hash.replace(/^#/, '');
  if (fromLink.startsWith('hwu_')) {
    storage.setItem(TOKEN_KEY, fromLink);
    replaceURL(path);
    return fromLink;
  }
  return storage.getItem(TOKEN_KEY);
}

export class CaptureError extends Error {
  constructor(
    message: string,
    readonly expired: boolean,
  ) {
    super(message);
  }
}

/**
 * Where the link adds photos: one inbox item when it was made on an item's
 * page (HW-IMP-10), otherwise new inbox items. Null when the link expired.
 */
export async function captureTarget(token: string, fetchFn: typeof fetch = fetch): Promise<{ item: boolean } | null> {
  const resp = await fetchFn('/api/capture', {
    headers: { Authorization: `Bearer ${token}` },
    credentials: 'omit',
  });
  if (resp.status === 401) return null;
  if (!resp.ok) throw new CaptureError(`The Mac did not answer (${resp.status})`, false);
  return (await resp.json()) as { item: boolean };
}

/** The message after a send: items made, or files added to the link's item. */
export function sentMessage(res: { added: number; files: number; duplicates: number }, toItem: boolean): string {
  const dup = res.duplicates ? `; ${res.duplicates} already stored` : '';
  if (toItem) return `Sent: ${res.files} ${res.files === 1 ? 'page' : 'pages'} added to the item on the Mac${dup}.`;
  return `Sent: ${res.added} ${res.added === 1 ? 'item' : 'items'} in the inbox on the Mac${dup}.`;
}

/**
 * Uploads one photo for Send with the link's token, without cookies, with
 * its progress (HW-IMP-11).
 */
export async function stageFromPhone(
  token: string,
  file: File,
  onProgress: (fraction: number) => void,
  xhr?: () => XMLHttpRequest,
): Promise<{ fileId: string; duplicate: boolean }> {
  const form = new FormData();
  form.append('files', file, file.name);
  const res = await postForm('/api/uploads', form, {
    headers: { Authorization: `Bearer ${token}` },
    withCredentials: false,
    onProgress,
    xhr,
  }).catch((error: unknown) => {
    throw new CaptureError(`The Mac did not answer. Check that Tailscale is on. (${String(error)})`, false);
  });
  if (res.status === 401) throw new CaptureError('This link has expired. Make a new QR code on the Mac.', true);
  if (res.status < 200 || res.status >= 300) {
    throw new CaptureError(
      typeof res.body.error === 'string' ? res.body.error : `Upload failed (${res.status})`,
      false,
    );
  }
  const [first] = res.body.files as { file: { id: string }; duplicate: boolean }[];
  return { fileId: first.file.id, duplicate: first.duplicate };
}

/** Files the photos sent ahead, by id, with the link's token, without cookies. */
export async function sendFromPhone(
  token: string,
  fileIds: string[],
  oneItem: boolean,
  fetchFn: typeof fetch = fetch,
): Promise<{ added: number; files: number; duplicates: number }> {
  const form = new FormData();
  for (const id of fileIds) form.append('fileId', id);
  if (oneItem && fileIds.length > 1) form.set('oneItem', 'true');
  let resp: Response;
  try {
    resp = await fetchFn('/api/inbox', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
      credentials: 'omit',
    });
  } catch (error) {
    throw new CaptureError(`The Mac did not answer. Check that Tailscale is on. (${String(error)})`, false);
  }
  if (resp.status === 401) throw new CaptureError('This link has expired. Make a new QR code on the Mac.', true);
  const body = (await resp.json().catch(() => ({}))) as {
    error?: string;
    items?: unknown[];
    files?: { duplicate: boolean }[];
  };
  if (!resp.ok) throw new CaptureError(body.error ?? `Upload failed (${resp.status})`, false);
  const duplicates = body.files?.filter((f) => f.duplicate).length ?? 0;
  return { added: body.items?.length ?? 0, files: (body.files?.length ?? 0) - duplicates, duplicates };
}
