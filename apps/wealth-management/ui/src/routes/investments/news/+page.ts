import { loadAIStatus } from '$lib/ai/aiStatus';
import type { PageLoad } from './$types';

// The AI status streams in: the page and its controls show at once, and the
// reason AI cannot run (or the status error) appears above them when known.
export const load: PageLoad = ({ fetch }) => ({ aiStatus: loadAIStatus(fetch) });
