// Stands in for SvelteKit's $app/navigation in this library's tests (vite.config.mjs).
export function invalidateAll(): Promise<void> {
  return Promise.resolve();
}
