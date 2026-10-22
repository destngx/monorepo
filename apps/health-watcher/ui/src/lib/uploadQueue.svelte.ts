/**
 * The phone camera's photos (HW-IMP-11). They stay on the phone until Send,
 * which uploads them a few at a time, each with its own progress, then names
 * the files now on the Mac. A failed photo is sent again on the next Send.
 */
export type StageFn = (
  file: File,
  onProgress: (fraction: number) => void,
) => Promise<{ fileId: string; duplicate: boolean }>;

/** waiting: on the phone only; queued: Send pressed, its turn to upload is coming. */
export type EntryState = 'waiting' | 'queued' | 'sending' | 'sent' | 'duplicate' | 'failed';

export interface QueueEntry {
  key: number;
  file: File;
  url: string;
  state: EntryState;
  /** Uploaded share of the file, 0 to 1. */
  progress: number;
  fileId?: string;
  error?: string;
}

export class UploadQueue {
  entries = $state<QueueEntry[]>([]);
  readonly #stage: StageFn;
  readonly #limit: number;
  readonly #makeURL: (f: File) => string;
  readonly #revokeURL: (url: string) => void;
  #next = 0;
  #waiters: (() => void)[] = [];

  constructor(
    stage: StageFn,
    options: { limit?: number; makeURL?: (f: File) => string; revokeURL?: (url: string) => void } = {},
  ) {
    this.#stage = stage;
    this.#limit = options.limit ?? 2;
    this.#makeURL = options.makeURL ?? ((f) => URL.createObjectURL(f));
    this.#revokeURL = options.revokeURL ?? ((u) => URL.revokeObjectURL(u));
  }

  add(files: File[]): void {
    for (const file of files) {
      this.entries.push({ key: this.#next++, file, url: this.#makeURL(file), state: 'waiting', progress: 0 });
    }
  }

  /** Uploads the photos not on the Mac yet, failed ones again; resolves when none is left uploading. */
  upload(): Promise<void> {
    for (const e of this.entries) {
      if (e.state !== 'failed') continue;
      e.state = 'waiting';
      e.progress = 0;
      e.error = undefined;
    }
    for (const e of this.entries) if (e.state === 'waiting') e.state = 'queued';
    const done = new Promise<void>((resolve) => this.#waiters.push(resolve));
    this.#pump();
    return done;
  }

  remove(key: number): void {
    const e = this.entries.find((x) => x.key === key);
    if (!e) return;
    this.#revokeURL(e.url);
    this.entries = this.entries.filter((x) => x.key !== key);
  }

  clear(): void {
    for (const e of this.entries) this.#revokeURL(e.url);
    this.entries = [];
  }

  /** Photos Send is still uploading. */
  get pending(): number {
    return this.entries.filter((e) => e.state === 'queued' || e.state === 'sending').length;
  }

  get failed(): number {
    return this.entries.filter((e) => e.state === 'failed').length;
  }

  /** The files on the Mac, in the order taken; duplicates are left out. */
  get fileIds(): string[] {
    return this.entries.filter((e) => e.state === 'sent' && e.fileId).map((e) => e.fileId!);
  }

  get duplicates(): number {
    return this.entries.filter((e) => e.state === 'duplicate').length;
  }

  #pump(): void {
    let running = this.entries.filter((e) => e.state === 'sending').length;
    for (const e of this.entries) {
      if (running >= this.#limit) break;
      if (e.state !== 'queued') continue;
      running++;
      void this.#run(e);
    }
    if (this.pending === 0) for (const resolve of this.#waiters.splice(0)) resolve();
  }

  async #run(e: QueueEntry): Promise<void> {
    e.state = 'sending';
    try {
      const res = await this.#stage(e.file, (p) => (e.progress = p));
      e.fileId = res.fileId;
      e.progress = 1;
      e.state = res.duplicate ? 'duplicate' : 'sent';
    } catch (error) {
      e.state = 'failed';
      e.error = error instanceof Error ? error.message : String(error);
    }
    // A removed photo finishes quietly.
    this.#pump();
  }
}
