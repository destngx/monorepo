import { site } from '$lib/data/site';
import { buildRobots } from '$lib/feeds';
import type { RequestHandler } from './$types';

export const prerender = true;

export const GET: RequestHandler = () =>
  new Response(buildRobots(site.url), { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
