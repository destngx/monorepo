// Polyfill for Node 20 compatibility
import util from 'util';

if (typeof globalThis !== 'undefined' && !(util as any).styleText) {
  // Add a minimal styleText implementation for test environments that expect it.
  // Keep this implementation intentionally simple.
  (util as any).styleText = (format: string, text: string) => text;
}
