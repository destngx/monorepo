/**
 * Files waiting to be sent, kept in the browser's IndexedDB. On an iPhone,
 * Safari may unload the page while the camera is open, and the dev server
 * reloads it when its connection drops; a draft brings the photos back.
 */
export interface Draft {
  files: File[];
  oneItem: boolean;
  note: string;
  personId: string;
}

/** A file as stored: IndexedDB keeps the bytes as a Blob. */
interface StoredFile {
  name: string;
  type: string;
  lastModified: number;
  blob: Blob;
}

interface StoredDraft extends Omit<Draft, 'files'> {
  files: StoredFile[];
}

export interface DraftBackend {
  get(key: string): Promise<StoredDraft | undefined>;
  set(key: string, value: StoredDraft): Promise<void>;
  delete(key: string): Promise<void>;
}

export function memoryBackend(): DraftBackend {
  const m = new Map<string, StoredDraft>();
  return {
    get: (k) => Promise.resolve(m.get(k)),
    set: (k, v) => {
      m.set(k, v);
      return Promise.resolve();
    },
    delete: (k) => {
      m.delete(k);
      return Promise.resolve();
    },
  };
}

const DB = 'health-watcher';
const STORE = 'drafts';

function promised<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error('IndexedDB request failed'));
  });
}

/** The IndexedDB backend, or null where the browser has none. */
export function idbBackend(): DraftBackend | null {
  if (typeof indexedDB === 'undefined') return null;
  let db: Promise<IDBDatabase> | null = null;
  const open = () => {
    db ??= new Promise((resolve, reject) => {
      const req = indexedDB.open(DB, 1);
      req.onupgradeneeded = () => req.result.createObjectStore(STORE);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error ?? new Error('Could not open IndexedDB'));
    });
    return db;
  };
  const store = async (mode: IDBTransactionMode) => (await open()).transaction(STORE, mode).objectStore(STORE);
  return {
    get: async (k) => (await promised((await store('readonly')).get(k))) as StoredDraft | undefined,
    set: async (k, v) => void (await promised((await store('readwrite')).put(v, k))),
    delete: async (k) => void (await promised((await store('readwrite')).delete(k))),
  };
}

export function draftStore(backend: DraftBackend | null) {
  return {
    async load(key: string): Promise<Draft | undefined> {
      const d = await backend?.get(key);
      if (!d?.files.length) return undefined;
      const files = d.files.map((f) => new File([f.blob], f.name, { type: f.type, lastModified: f.lastModified }));
      return { files, oneItem: d.oneItem, note: d.note, personId: d.personId };
    },
    async save(key: string, draft: Draft): Promise<void> {
      if (!backend) return;
      if (!draft.files.length) return backend.delete(key);
      const files = draft.files.map((f) => ({ name: f.name, type: f.type, lastModified: f.lastModified, blob: f }));
      await backend.set(key, { ...draft, files });
    },
    async clear(key: string): Promise<void> {
      await backend?.delete(key);
    },
  };
}
