// /capture is the phone camera page; its link carries an upload token instead of a session.
const PUBLIC = ['/login', '/setup', '/invite', '/capture'];

/** Pages reachable without a session (path without the /ui base). */
export function isPublicPath(path: string): boolean {
  return PUBLIC.some((p) => path === p || path.startsWith(`${p}/`));
}

/** A same-app path from a ?next= value; anything else falls back to home. */
export function safeNext(next: string | null): string {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : '/';
}
