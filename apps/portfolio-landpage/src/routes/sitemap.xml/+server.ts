import { site } from '$lib/data/site';
import { buildSitemap } from '$lib/feeds';
import { getPosts, getProjects } from '$lib/server/content';
import type { RequestHandler } from './$types';

export const prerender = true;

export const GET: RequestHandler = () => {
  const entries = [
    { path: '/' },
    { path: '/projects' },
    { path: '/blog' },
    ...getProjects().map((project) => ({ path: `/projects/${project.slug}` })),
    ...getPosts()
      .filter((post) => !post.draft)
      .map((post) => ({ path: `/blog/${post.slug}`, lastModified: post.updated ?? post.date })),
  ];

  return new Response(buildSitemap(site.url, entries), {
    headers: { 'Content-Type': 'application/xml; charset=utf-8' },
  });
};
