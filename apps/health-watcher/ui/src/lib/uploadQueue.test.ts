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
  it('uploads two at a time as photos arrive and is ready when all are up', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1), photo(2), photo(3)]);
    expect(m.calls.map((c) => c.name)).toEqual(['IMG_1.jpg', 'IMG_2.jpg']);
    expect(q.ready).toBe(false);

    m.calls[0].progress(0.5);
    expect(q.entries[0].progress).toBe(0.5);
    m.calls[0].done('f1');
    await tick();
    expect(m.calls.map((c) => c.name)).toEqual(['IMG_1.jpg', 'IMG_2.jpg', 'IMG_3.jpg']);

    m.calls[1].done('f2');
    m.calls[2].done('f3', true);
    await tick();
    expect(q.fileIds).toEqual(['f1', 'f2']);
    expect(q.duplicates).toBe(1);
    expect(q.ready).toBe(true);
  });

  it('holds Send while a photo failed, and retries it', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1)]);
    m.calls[0].fail('The Mac did not answer');
    await tick();
    expect(q.entries[0].state).toBe('failed');
    expect(q.entries[0].error).toBe('The Mac did not answer');
    expect(q.ready).toBe(false);

    q.retry(q.entries[0].key);
    expect(m.calls).toHaveLength(2);
    m.calls[1].done('f1');
    await tick();
    expect(q.ready).toBe(true);
  });

  it('drops a removed photo from what Send names', async () => {
    const m = manual();
    const q = new UploadQueue(m.stage, opts);
    q.add([photo(1), photo(2)]);
    m.calls[0].done('f1');
    m.calls[1].done('f2');
    await tick();
    q.remove(q.entries[0].key);
    expect(q.fileIds).toEqual(['f2']);
  });
});
