import { describe, expect, it } from 'vitest';
import { draftStore, memoryBackend } from './drafts';

describe('upload drafts', () => {
  it('given staged photos when the page reloads then they come back with the options', async () => {
    const store = draftStore(memoryBackend());
    const photo = new File(['jpeg bytes'], 'IMG_0001.jpg', { type: 'image/jpeg', lastModified: 1000 });
    await store.save('upload:t1', { files: [photo], oneItem: true, note: 'clinic', personId: 'p1' });

    const draft = await store.load('upload:t1');
    expect(draft?.oneItem).toBe(true);
    expect(draft?.note).toBe('clinic');
    expect(draft?.personId).toBe('p1');
    expect(draft?.files).toHaveLength(1);
    expect(draft?.files[0].name).toBe('IMG_0001.jpg');
    expect(draft?.files[0].type).toBe('image/jpeg');
    expect(await draft?.files[0].text()).toBe('jpeg bytes');
    expect(await store.load('upload:t2')).toBeUndefined();
  });

  it('given no staged photos when saving then the draft is removed', async () => {
    const store = draftStore(memoryBackend());
    await store.save('k', { files: [new File(['x'], 'a.jpg')], oneItem: false, note: '', personId: '' });
    await store.save('k', { files: [], oneItem: false, note: '', personId: '' });
    expect(await store.load('k')).toBeUndefined();
  });

  it('given no storage in the browser when used then nothing is kept and nothing fails', async () => {
    const store = draftStore(null);
    await store.save('k', { files: [new File(['x'], 'a.jpg')], oneItem: false, note: '', personId: '' });
    expect(await store.load('k')).toBeUndefined();
    await store.clear('k');
  });
});
