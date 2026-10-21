/** Upload rules the browser checks before sending (HW-IMP-2); the engine checks again. */
export const MAX_FILE_MB = 50;
export const MAX_BATCH_MB = 200;
export const MAX_FILES = 60;

const ACCEPTED_EXT = /\.(pdf|jpe?g|png|webp|heic|heif)$/i;
const ACCEPTED_TYPES = new Set([
  'application/pdf',
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/heic',
  'image/heif',
]);

/** The file picker's accept list. */
export const ACCEPT = '.pdf,.jpg,.jpeg,.png,.webp,.heic,.heif,application/pdf,image/*';

export interface Picked {
  accepted: File[];
  /** Names left out, with why. */
  skipped: string[];
}

/** A file the engine accepts, by type or, when the browser gives none, by name. */
export function isAccepted(file: File): boolean {
  if (file.type) return ACCEPTED_TYPES.has(file.type);
  return ACCEPTED_EXT.test(file.name);
}

/**
 * Splits picked files into accepted and skipped: hidden files (.DS_Store
 * from a folder), other types, empty and oversized files, and anything past
 * the batch limits.
 */
export function pickFiles(files: Iterable<File>, alreadyStaged: File[] = []): Picked {
  const accepted: File[] = [];
  const skipped: string[] = [];
  let total = alreadyStaged.reduce((sum, f) => sum + f.size, 0);
  let count = alreadyStaged.length;
  for (const file of files) {
    if (file.name.startsWith('.')) continue;
    if (!isAccepted(file)) {
      skipped.push(`${file.name} (not a PDF or image)`);
    } else if (file.size === 0) {
      skipped.push(`${file.name} (empty)`);
    } else if (file.size > MAX_FILE_MB * 1024 * 1024) {
      skipped.push(`${file.name} (over ${MAX_FILE_MB} MB)`);
    } else if (count >= MAX_FILES || total + file.size > MAX_BATCH_MB * 1024 * 1024) {
      skipped.push(`${file.name} (over the ${MAX_FILES}-file or ${MAX_BATCH_MB} MB batch limit)`);
    } else {
      accepted.push(file);
      total += file.size;
      count++;
    }
  }
  return { accepted, skipped };
}
