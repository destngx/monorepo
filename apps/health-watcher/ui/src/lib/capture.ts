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

/** Sends the photos to the inbox with the link's token, without cookies. */
export async function sendFromPhone(
  token: string,
  files: File[],
  oneItem: boolean,
  fetchFn: typeof fetch = fetch,
): Promise<{ added: number; duplicates: number }> {
  const form = new FormData();
  for (const f of files) form.append('files', f, f.name);
  if (oneItem && files.length > 1) form.set('oneItem', 'true');
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
  return { added: body.items?.length ?? 0, duplicates: body.files?.filter((f) => f.duplicate).length ?? 0 };
}
