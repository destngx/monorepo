// Sections load independently, so one failing fmarket action never blanks the page.

export type Section<T> = { ok: true; value: T } | { ok: false; error: string };

export async function settle<T>(load: () => Promise<T>): Promise<Section<T>> {
  try {
    return { ok: true, value: await load() };
  } catch (cause) {
    return { ok: false, error: cause instanceof Error ? cause.message : String(cause) };
  }
}
