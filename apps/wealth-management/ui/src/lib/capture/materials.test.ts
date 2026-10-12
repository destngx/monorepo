import { describe, expect, it } from 'vitest';
import {
  batchUsage,
  fitWithin,
  formatBytes,
  isImageFile,
  nextMaterialId,
  textBytes,
  toBase64,
  type InboxItem,
} from './materials';

const limits = { maxItems: 2, maxItemBytes: 1000, maxBatchBytes: 1500 };
const item = (id: string, size: number): InboxItem => ({ id, kind: 'text', name: id, size, text: 'x' });

describe('nextMaterialId', () => {
  it('skips ids already in the inbox', () => {
    expect(nextMaterialId([])).toBe('m1');
    expect(nextMaterialId([{ id: 'm2' }])).toBe('m3');
    expect(nextMaterialId([{ id: 'm3' }, { id: 'm1' }])).toBe('m4');
  });
});

describe('batchUsage', () => {
  it('accepts a batch within the limits', () => {
    expect(batchUsage([item('a', 500), item('b', 900)], limits)).toEqual({ count: 2, bytes: 1400, problems: [] });
  });

  it('reports every limit the engine would enforce', () => {
    const usage = batchUsage([item('a', 1200), item('b', 400), item('c', 10)], limits);
    expect(usage.problems).toEqual([
      'At most 2 materials per batch.',
      'a is 1 KB; the limit is 1000 B.',
      'The batch is 2 KB; the limit is 1 KB.',
    ]);
  });
});

describe('image helpers', () => {
  it('fits the longest edge without upscaling', () => {
    expect(fitWithin(1170, 2532, 2048)).toEqual({ width: 946, height: 2048 });
    expect(fitWithin(800, 600, 2048)).toEqual({ width: 800, height: 600 });
  });

  it('treats HEIC files as images even without a MIME type', () => {
    expect(isImageFile({ type: '', name: 'IMG_0001.HEIC' })).toBe(true);
    expect(isImageFile({ type: 'image/png', name: 'a.png' })).toBe(true);
    expect(isImageFile({ type: 'text/plain', name: 'a.txt' })).toBe(false);
  });

  it('encodes bytes as base64 across chunk boundaries', () => {
    const bytes = new Uint8Array(70000).map((_, i) => i % 256);
    expect(toBase64(bytes)).toBe(Buffer.from(bytes).toString('base64'));
  });
});

describe('sizes', () => {
  it('counts UTF-8 bytes like the engine', () => {
    expect(textBytes('đ')).toBe(2);
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB');
  });
});
