import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { ApiError } from '$lib/api/client';
import { auth } from '$lib/api/endpoints';
import { isPublicPath } from '$lib/paths';
import { session } from '$lib/state/session.svelte';
import type { LayoutLoad } from './$types';

// Client-rendered SPA: the Go engine serves the static shell and the browser does the rest.
export const ssr = false;

// Runs once per full load (the path is read untracked), and again after invalidateAll.
export const load: LayoutLoad = async ({ fetch, url, untrack }) => {
  const path = untrack(() => url.pathname.slice(base.length) || '/');
  const search = untrack(() => url.search);
  try {
    session.set(await auth.me(fetch));
    return {};
  } catch (error) {
    if (!(error instanceof ApiError && error.status === 401)) throw error;
  }
  session.clear();
  if (isPublicPath(path)) return {};
  const status = await auth.setupStatus(fetch);
  if (status.needsSetup) redirect(307, `${base}/setup`);
  redirect(307, `${base}/login?next=${encodeURIComponent(path + search)}`);
};
