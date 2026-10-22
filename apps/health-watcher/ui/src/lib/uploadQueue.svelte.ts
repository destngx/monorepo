/**
 * The phone's send-ahead queue (HW-IMP-11): each photo starts uploading as
 * soon as it is taken, a few at a time, so Send only has to name the files
 * already on the Mac. Each photo has its own progress.
 */
export type StageFn = (
  file: File,
  onProgress: (fraction: number) => void,
) => Promise<{ fileId: string; duplicate: boolean }>;

export type EntryState = 'waiting' | 'sending' | 'sent' | 'duplicate' | 'failed';

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
    this.#pump();
  }

  retry(key: number): void {
    const e = this.entries.find((x) => x.key === key);
    if (!e || e.state !== 'failed') return;
    e.state = 'waiting';
    e.progress = 0;
    e.error = undefined;
    this.#pump();
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

  /** Photos still waiting or uploading. */
  get pending(): number {
    return this.entries.filter((e) => e.state === 'waiting' || e.state === 'sending').length;
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

  /** Send may go once every photo is up, none failed and one is new. */
  get ready(): boolean {
    return this.pending === 0 && this.failed === 0 && this.fileIds.length > 0;
  }

  #pump(): void {
    let running = this.entries.filter((e) => e.state === 'sending').length;
    for (const e of this.entries) {
      if (running >= this.#limit) return;
      if (e.state !== 'waiting') continue;
      running++;
      void this.#run(e);
    }
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
