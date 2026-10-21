import { describe, expect, it } from 'vitest';
import { isAccepted, pickFiles, MAX_FILES } from './upload';

function file(name: string, type: string, size = 10): File {
  return new File([new Uint8Array(size)], name, { type });
}

describe('isAccepted', () => {
  it('accepts PDFs and images by type, or by name when the type is missing', () => {
    expect(isAccepted(file('a.pdf', 'application/pdf'))).toBe(true);
    expect(isAccepted(file('IMG_1.HEIC', ''))).toBe(true);
    expect(isAccepted(file('notes.txt', 'text/plain'))).toBe(false);
    expect(isAccepted(file('page.html', ''))).toBe(false);
  });
});

describe('pickFiles', () => {
  it('drops hidden files silently and explains the rest', () => {
    const { accepted, skipped } = pickFiles([
      file('.DS_Store', ''),
      file('scan.pdf', 'application/pdf'),
      file('empty.jpg', 'image/jpeg', 0),
      file('notes.txt', 'text/plain'),
    ]);
    expect(accepted.map((f) => f.name)).toEqual(['scan.pdf']);
    expect(skipped).toEqual(['empty.jpg (empty)', 'notes.txt (not a PDF or image)']);
  });

  it('stops at the file count, counting files already staged', () => {
    const staged = Array.from({ length: MAX_FILES - 1 }, (_, i) => file(`p${i}.jpg`, 'image/jpeg'));
    const { accepted, skipped } = pickFiles([file('a.jpg', 'image/jpeg'), file('b.jpg', 'image/jpeg')], staged);
    expect(accepted).toHaveLength(1);
    expect(skipped[0]).toContain('b.jpg');
  });
});
