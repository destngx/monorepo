import { describe, expect, it } from 'vitest';
import { buildRss, buildSitemap, escapeXml } from './feeds';
import type { Post } from './content/schema';

const site = { url: 'https://example.dev', title: 'Example', description: 'Desc' };

describe('escapeXml', () => {
  it('escapes the five XML special characters', () => {
    expect(escapeXml(`<a href="x">Tom & 'Jerry'</a>`)).toBe(
      '&lt;a href=&quot;x&quot;&gt;Tom &amp; &apos;Jerry&apos;&lt;/a&gt;',
    );
  });
});

describe('buildRss', () => {
  it('renders an RSS 2.0 channel with one escaped item per post', () => {
    const posts = [
      { slug: 'ci-tips', title: 'CI & CD', description: 'Fast <pipelines>', date: new Date('2026-01-02T00:00:00Z') },
    ] as Post[];

    const xml = buildRss(posts, site);

    expect(xml).toMatch(/^<\?xml version="1.0" encoding="UTF-8"\?>/);
    expect(xml).toContain('<link>https://example.dev/blog/ci-tips</link>');
    expect(xml).toContain('<title>CI &amp; CD</title>');
    expect(xml).toContain('<description>Fast &lt;pipelines&gt;</description>');
    expect(xml).toContain('<pubDate>Fri, 02 Jan 2026 00:00:00 GMT</pubDate>');
  });
});

describe('buildSitemap', () => {
  it('lists absolute URLs with optional lastmod', () => {
    const xml = buildSitemap(site.url, [
      { path: '/' },
      { path: '/blog/a', lastModified: new Date('2026-05-01T00:00:00Z') },
    ]);

    expect(xml).toContain('<loc>https://example.dev/</loc>');
    expect(xml).toContain('<loc>https://example.dev/blog/a</loc><lastmod>2026-05-01</lastmod>');
  });
});
