import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { LayoutLoad } from './$types';

// Client-rendered SPA: the Go engine serves the static shell and the browser does the rest.
export const ssr = false;

// The health check streams in: pages never wait for it. An unreachable
// engine resolves to a failure, which the layout shows as a banner.
export const load: LayoutLoad = ({ fetch }) => ({ engine: defer(() => api.health(fetch)) });
