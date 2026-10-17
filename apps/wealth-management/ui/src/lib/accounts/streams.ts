import type { Streamed } from '$lib/api/load';

/**
 * Two streamed requests as one, for a section that needs both (the balance
 * history needs the histories and the accounts): both values, or the first
 * failure. Like `defer`, it never rejects.
 */
export async function joinStreamed<A, B>(a: Streamed<A>, b: Streamed<B>): Streamed<[A, B]> {
  const [left, right] = await Promise.all([a, b]);
  if (!left.ok) return left;
  if (!right.ok) return right;
  return { ok: true, value: [left.value, right.value] };
}
