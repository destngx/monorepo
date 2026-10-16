import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [bills, idleCash] = await Promise.all([api.bills({ fetch }), api.idleCash({ fetch })]);
    return { bills, idleCash };
  });
