// Lets views react to finished AI requests (each one may spend provider
// quota) without the API layer depending on them.
type Listener = () => void;

const listeners = new Set<Listener>();

/** Calls `listener` after every AI request that may have reached the model; returns the unsubscribe. */
export function onAIRequestDone(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function aiRequestDone(): void {
  for (const listener of listeners) listener();
}
