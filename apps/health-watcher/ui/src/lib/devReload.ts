/**
 * Keeps the Vite dev client from reloading the page when its connection to
 * the dev server drops. iOS drops it each time the camera opens, so the
 * phone camera pages would reload after every photo. The client waits for
 * every disconnect listener before it reloads, and this one never settles.
 * The page loses hot updates until it is reloaded by hand, which is fine on
 * a phone. Does nothing outside the dev server. Returns the undo.
 */
export function holdDevReload(): () => void {
  const hot = import.meta.hot;
  if (!hot) return () => undefined;
  const hold = () => new Promise<never>(() => undefined);
  hot.on('vite:ws:disconnect', hold);
  return () => hot.off('vite:ws:disconnect', hold);
}
