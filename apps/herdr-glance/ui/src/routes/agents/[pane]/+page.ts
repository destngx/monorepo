import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => ({ paneId: params.pane });
