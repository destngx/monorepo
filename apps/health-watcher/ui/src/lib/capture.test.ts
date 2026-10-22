import { describe, expect, it, vi } from 'vitest';
import { CaptureError, captureTarget, sendFromPhone, sentMessage, stageFromPhone, takeCaptureToken } from './capture';

function storage(): Storage {
  const m = new Map<string, string>();
  return {
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => void m.set(k, v),
    removeItem: (k) => void m.delete(k),
    clear: () => m.clear(),
    key: () => null,
    get length() {
      return m.size;
    },
  };
}

describe('phone capture link', () => {
  it('given a link with a token when opened then it is kept for reloads and leaves the address bar', () => {
    const store = storage();
    const replace = vi.fn();
    expect(takeCaptureToken('#hwu_abc', '/ui/capture', store, replace)).toBe('hwu_abc');
    expect(replace).toHaveBeenCalledWith('/ui/capture');
    // After a reload the fragment is gone; the kept token is used.
    expect(takeCaptureToken('', '/ui/capture', store, replace)).toBe('hwu_abc');
  });

  it('given a link without a token when opened then there is none', () => {
    expect(takeCaptureToken('#nonsense', '/ui/capture', storage(), vi.fn())).toBeNull();
  });
});

describe('sending from the phone', () => {
  it('given photos sent ahead when Send is pressed then their ids go to the inbox with the token and no cookies', async () => {
    const fetchFn = vi.fn<(url: string, init: RequestInit) => Promise<Response>>(() =>
      Promise.resolve(
        Response.json({ items: [{}], files: [{ duplicate: false }, { duplicate: true }] }, { status: 202 }),
      ),
    );
    const res = await sendFromPhone('hwu_abc', ['f1', 'f2'], true, fetchFn as unknown as typeof fetch);
    expect(res).toEqual({ added: 1, files: 1, duplicates: 1 });
    const [url, init] = fetchFn.mock.calls[0];
    expect(url).toBe('/api/inbox');
    expect(init.credentials).toBe('omit');
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer hwu_abc');
    const form = init.body as FormData;
    expect(form.getAll('fileId')).toEqual(['f1', 'f2']);
    expect(form.get('oneItem')).toBe('true');
  });

  it('given an expired link when sending then it says so', async () => {
    const fetchFn = vi.fn(() => Promise.resolve(Response.json({ error: 'Not signed in.' }, { status: 401 })));
    await expect(sendFromPhone('hwu_abc', ['f1'], false, fetchFn as unknown as typeof fetch)).rejects.toMatchObject({
      expired: true,
    });
  });

  it('given a refused upload when sending then the engine message comes back', async () => {
    const fetchFn = vi.fn(() =>
      Promise.resolve(Response.json({ error: 'IMG_0001.jpg is larger than 50 MB' }, { status: 400 })),
    );
    const err: unknown = await sendFromPhone('hwu_abc', ['f1'], false, fetchFn as unknown as typeof fetch).catch(
      (e: unknown) => e,
    );
    expect(err).toBeInstanceOf(CaptureError);
    expect((err as CaptureError).message).toBe('IMG_0001.jpg is larger than 50 MB');
    expect((err as CaptureError).expired).toBe(false);
  });
});

describe('the link target', () => {
  it('given a link made on an item when asked then it says so, with the token and no cookies', async () => {
    const fetchFn = vi.fn<(url: string, init: RequestInit) => Promise<Response>>(() =>
      Promise.resolve(Response.json({ item: true })),
    );
    expect(await captureTarget('hwu_abc', fetchFn as unknown as typeof fetch)).toEqual({ item: true });
    const [url, init] = fetchFn.mock.calls[0];
    expect(url).toBe('/api/capture');
    expect(init.credentials).toBe('omit');
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer hwu_abc');
  });

  it('given an expired link when asked then there is no target', async () => {
    const fetchFn = vi.fn(() => Promise.resolve(Response.json({ error: 'Not signed in.' }, { status: 401 })));
    expect(await captureTarget('hwu_abc', fetchFn as unknown as typeof fetch)).toBeNull();
  });

  it('given a send when reporting then an item link counts pages and an inbox link counts items', () => {
    expect(sentMessage({ added: 1, files: 2, duplicates: 1 }, true)).toBe(
      'Sent: 2 pages added to the item on the Mac; 1 already stored.',
    );
    expect(sentMessage({ added: 2, files: 2, duplicates: 0 }, false)).toBe('Sent: 2 items in the inbox on the Mac.');
  });
});

/** A fake XMLHttpRequest that answers with a status and body after reporting progress. */
function fakeXHR(status: number, body: unknown) {
  const sent: { url?: string; headers: Record<string, string>; withCredentials?: boolean; form?: FormData } = {
    headers: {},
  };
  const make = () => {
    const x = {
      upload: { onprogress: null as ((e: ProgressEvent) => void) | null },
      onload: null as (() => void) | null,
      onerror: null as (() => void) | null,
      onabort: null as (() => void) | null,
      status: 0,
      responseText: '',
      withCredentials: false,
      open: (_m: string, url: string) => (sent.url = url),
      setRequestHeader: (k: string, v: string) => (sent.headers[k] = v),
      send: (form: FormData) => {
        sent.form = form;
        sent.withCredentials = x.withCredentials;
        x.upload.onprogress?.({ lengthComputable: true, loaded: 50, total: 100 } as ProgressEvent);
        x.status = status;
        x.responseText = JSON.stringify(body);
        x.onload?.();
      },
    };
    return x as unknown as XMLHttpRequest;
  };
  return { make, sent };
}

describe('sending a photo ahead', () => {
  const photo = new File(['jpeg'], 'IMG_0001.jpg', { type: 'image/jpeg' });

  it('given a photo when sent ahead then progress is reported and the file id comes back', async () => {
    const x = fakeXHR(202, { files: [{ file: { id: 'f1' }, duplicate: false }] });
    const seen: number[] = [];
    const res = await stageFromPhone('hwu_abc', photo, (p) => seen.push(p), x.make);
    expect(res).toEqual({ fileId: 'f1', duplicate: false });
    expect(seen).toEqual([0.5]);
    expect(x.sent.url).toBe('/api/uploads');
    expect(x.sent.withCredentials).toBe(false);
    expect(x.sent.headers.Authorization).toBe('Bearer hwu_abc');
  });

  it('given an expired link when sending ahead then it says so', async () => {
    const x = fakeXHR(401, { error: 'Not signed in.' });
    await expect(stageFromPhone('hwu_abc', photo, vi.fn(), x.make)).rejects.toMatchObject({ expired: true });
  });
});
