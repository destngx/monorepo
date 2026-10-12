import { CAPTURE_IMAGE_TYPES, type CaptureLimits, type CaptureMaterialPayload } from '$lib/api/types';

// Smart Capture inbox (WM-CAP-1). Materials stay in this tab's memory only
// and are dropped after commit or when the page closes (WM-CAP-10).

export interface InboxItem {
  id: string;
  kind: 'image' | 'text';
  name: string;
  size: number;
  mediaType?: string;
  blob?: Blob;
  /** Object URL for the thumbnail; revoke it with releaseItem. */
  previewUrl?: string;
  text?: string;
}

/** Longest image edge sent to the model: keeps statement text legible at a fraction of the bytes. */
export const MAX_IMAGE_EDGE = 2048;

export function nextMaterialId(items: { id: string }[]): string {
  const used = new Set(items.map((item) => item.id));
  for (let n = items.length + 1; ; n++) {
    if (!used.has(`m${n}`)) return `m${n}`;
  }
}

/** UTF-8 byte length, the size the engine counts for text. */
export function textBytes(text: string): number {
  return new TextEncoder().encode(text).length;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export interface BatchUsage {
  count: number;
  bytes: number;
  problems: string[];
}

/** Mirrors the engine's batch validation so problems show before upload. */
export function batchUsage(items: InboxItem[], limits: CaptureLimits): BatchUsage {
  const bytes = items.reduce((sum, item) => sum + item.size, 0);
  const problems: string[] = [];
  if (items.length > limits.maxItems) problems.push(`At most ${limits.maxItems} materials per batch.`);
  for (const item of items) {
    if (item.size > limits.maxItemBytes) {
      problems.push(`${item.name} is ${formatBytes(item.size)}; the limit is ${formatBytes(limits.maxItemBytes)}.`);
    }
  }
  if (bytes > limits.maxBatchBytes) {
    problems.push(`The batch is ${formatBytes(bytes)}; the limit is ${formatBytes(limits.maxBatchBytes)}.`);
  }
  return { count: items.length, bytes, problems };
}

/** Scales width x height down (never up) so the longest edge is at most maxEdge. */
export function fitWithin(width: number, height: number, maxEdge: number): { width: number; height: number } {
  const scale = Math.min(1, maxEdge / Math.max(width, height));
  return { width: Math.max(1, Math.round(width * scale)), height: Math.max(1, Math.round(height * scale)) };
}

export function isSupportedImageType(type: string): boolean {
  return (CAPTURE_IMAGE_TYPES as readonly string[]).includes(type);
}

export function isImageFile(file: { type: string; name: string }): boolean {
  return file.type.startsWith('image/') || /\.(heic|heif)$/i.test(file.name);
}

export class UnreadableImageError extends Error {
  constructor(name: string) {
    super(
      `${name} could not be read by this browser. HEIC photos only open in Safari: convert to JPEG or take a screenshot instead.`,
    );
    this.name = 'UnreadableImageError';
  }
}

/**
 * Returns an image the engine accepts within the item limit: the original
 * when it already fits, else a downscaled JPEG (HEIC and other formats are
 * converted the same way when the browser can decode them).
 */
export async function prepareImage(
  file: File,
  maxItemBytes: number,
  maxEdge = MAX_IMAGE_EDGE,
): Promise<{ blob: Blob; mediaType: string }> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file);
  } catch {
    throw new UnreadableImageError(file.name || 'The image');
  }
  try {
    const isWithinLimits = Math.max(bitmap.width, bitmap.height) <= maxEdge && file.size <= maxItemBytes;
    if (isWithinLimits && isSupportedImageType(file.type)) return { blob: file, mediaType: file.type };

    let { width, height } = fitWithin(bitmap.width, bitmap.height, maxEdge);
    for (let attempt = 0; attempt < 5; attempt++) {
      const blob = await encodeJPEG(bitmap, width, height, attempt === 0 ? 0.9 : 0.8);
      if (blob.size <= maxItemBytes) return { blob, mediaType: 'image/jpeg' };
      width = Math.round(width * 0.75);
      height = Math.round(height * 0.75);
    }
    throw new Error(`${file.name || 'The image'} is still over the size limit after downscaling.`);
  } finally {
    bitmap.close();
  }
}

async function encodeJPEG(bitmap: ImageBitmap, width: number, height: number, quality: number): Promise<Blob> {
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext('2d');
  if (!context) throw new Error('This browser cannot resize images.');
  // JPEG has no alpha: paint white under transparent screenshots.
  context.fillStyle = '#fff';
  context.fillRect(0, 0, width, height);
  context.drawImage(bitmap, 0, 0, width, height);
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('Could not encode the image.'))),
      'image/jpeg',
      quality,
    ),
  );
}

export function toBase64(bytes: Uint8Array): string {
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

export async function toPayload(item: InboxItem): Promise<CaptureMaterialPayload> {
  if (item.kind === 'text') return { id: item.id, kind: 'text', text: item.text ?? '' };
  const bytes = new Uint8Array(await item.blob!.arrayBuffer());
  return { id: item.id, kind: 'image', name: item.name, mediaType: item.mediaType, data: toBase64(bytes) };
}

export function releaseItem(item: InboxItem): void {
  if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
}
