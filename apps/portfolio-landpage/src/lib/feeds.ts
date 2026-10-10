import type { Post } from './content/schema';

type FeedSite = { url: string; title: string; description: string };
export type SitemapEntry = { path: string; lastModified?: Date };

const XML_ENTITIES: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' };

export const escapeXml = (value: string): string => value.replace(/[&<>"']/g, (char) => XML_ENTITIES[char]);

export function buildRss(posts: Post[], site: FeedSite): string {
  const items = posts
    .map((post) => {
      const url = new URL(`/blog/${post.slug}`, site.url).href;
      return [
        '<item>',
        `<title>${escapeXml(post.title)}</title>`,
        `<link>${url}</link>`,
        `<guid isPermaLink="true">${url}</guid>`,
        `<description>${escapeXml(post.description)}</description>`,
        `<pubDate>${post.date.toUTCString()}</pubDate>`,
        '</item>',
      ].join('');
    })
    .join('');

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel>',
    `<title>${escapeXml(site.title)}</title>`,
    `<link>${site.url}</link>`,
    `<description>${escapeXml(site.description)}</description>`,
    `<atom:link href="${new URL('/rss.xml', site.url).href}" rel="self" type="application/rss+xml"/>`,
    items,
    '</channel></rss>',
  ].join('');
}

export function buildSitemap(baseUrl: string, entries: SitemapEntry[]): string {
  const urls = entries
    .map(({ path, lastModified }) => {
      const lastmod = lastModified ? `<lastmod>${lastModified.toISOString().slice(0, 10)}</lastmod>` : '';
      return `<url><loc>${new URL(path, baseUrl).href}</loc>${lastmod}</url>`;
    })
    .join('');

  return `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls}</urlset>`;
}
