import { describe, expect, it } from 'vitest';
import { UploadQueue, type StageFn } from './uploadQueue.svelte';

const photo = (n: number) => new File(['jpeg'], `IMG_${n}.jpg`, { type: 'image/jpeg' });

/** A stage function whose uploads finish when the test says so. */
function manual() {
  const calls: {
    name: string;
    progress: (p: number) => void;
    done: (id: string, dup?: boolean) => void;
    fail: (m: string) => void;
  }[] = [];
  const stage: StageFn = (file, progress) =>
    new Promise((resolve, reject) => {
      calls.push({
        name: file.name,
        progress,
        done: (id, dup = false) => resolve({ fileId: id, duplicate: dup }),
        fail: (m) => reject(new Error(m)),
      });
    });
  return { stage, calls };
}

const tick = () => new Promise((r) => setTimeout(r, 0));
const opts = { makeURL: (f: File) => `blob:${f.name}`, revokeURL: () => undefined };

describe('UploadQueue', () => {
  it('keeps photos on the phone until upload, then sends two at a time', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1), photo(2), photo(3)]);
    expect(m.calls).toHaveLength(0);
    expect(q.pending).toBe(0);

    const done = q.upload();
    expect(m.calls.map((c) => c.name)).toEqual(['IMG_1.jpg', 'IMG_2.jpg']);
    expect(q.pending).toBe(3);

    m.calls[0].progress(0.5);
    expect(q.entries[0].progress).toBe(0.5);
    m.calls[0].done('f1');
    await tick();
    expect(m.calls.map((c) => c.name)).toEqual(['IMG_1.jpg', 'IMG_2.jpg', 'IMG_3.jpg']);

    m.calls[1].done('f2');
    m.calls[2].done('f3', true);
    await done;
    expect(q.fileIds).toEqual(['f1', 'f2']);
    expect(q.duplicates).toBe(1);
    expect(q.pending).toBe(0);
  });

  it('sends only the failed photos again on the next upload', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1), photo(2)]);
    const first = q.upload();
    m.calls[0].done('f1');
    m.calls[1].fail('The Mac did not answer');
    await first;
    expect(q.entries[1].state).toBe('failed');
    expect(q.entries[1].error).toBe('The Mac did not answer');
    expect(q.failed).toBe(1);

    const again = q.upload();
    expect(m.calls.map((c) => c.name)).toEqual(['IMG_1.jpg', 'IMG_2.jpg', 'IMG_2.jpg']);
    m.calls[2].done('f2');
    await again;
    expect(q.failed).toBe(0);
    expect(q.fileIds).toEqual(['f1', 'f2']);
  });

  it('finishes at once when there is nothing to send', async () => {
    const q = new UploadQueue(manual().stage, opts);
    await q.upload();
    expect(q.pending).toBe(0);
  });

  it('drops a removed photo from what Send names', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1), photo(2)]);
    const done = q.upload();
    m.calls[0].done('f1');
    m.calls[1].done('f2');
    await done;
    q.remove(q.entries[0].key);
    expect(q.fileIds).toEqual(['f2']);
  });
});
