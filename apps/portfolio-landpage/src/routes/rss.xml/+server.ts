import { site } from '$lib/data/site';
import { buildRss } from '$lib/feeds';
import { getPosts } from '$lib/server/content';
import type { RequestHandler } from './$types';

export const prerender = true;

export const GET: RequestHandler = () =>
  new Response(
    buildRss(
      getPosts().filter((post) => !post.draft),
      site,
    ),
    {
      headers: { 'Content-Type': 'application/rss+xml; charset=utf-8' },
    },
  );
