import { describe, expect, it } from 'vitest';
import { parseCollection, publishedPosts, sortPosts, sortProjects } from './collection';
import { postMetaSchema, projectMetaSchema, type Post, type Project } from './schema';

const project = (overrides: Record<string, unknown> = {}) => ({
  metadata: {
    title: 'GraphWeave',
    summary: 'Workflow engine.',
    year: 2026,
    status: 'active',
    stack: ['Go'],
    featured: true,
    order: 1,
    ...overrides,
  },
});

describe('parseCollection', () => {
  it('derives the slug from the file name', () => {
    const items = parseCollection({ '/src/content/projects/graph-weave.md': project() }, projectMetaSchema);

    expect(items).toHaveLength(1);
    expect(items[0].slug).toBe('graph-weave');
    expect(items[0].title).toBe('GraphWeave');
  });

  it('fails loudly with the file path when frontmatter is invalid', () => {
    expect(() =>
      parseCollection({ '/src/content/projects/broken.md': project({ status: 'nope' }) }, projectMetaSchema),
    ).toThrow(/broken\.md/);
  });

  it('fails when a module has no frontmatter', () => {
    expect(() => parseCollection({ '/src/content/projects/empty.md': {} }, projectMetaSchema)).toThrow(/empty\.md/);
  });
});

describe('postMetaSchema', () => {
  it('coerces the date string into a Date and defaults optional fields', () => {
    const meta = postMetaSchema.parse({ title: 'Hi', description: 'First', date: '2026-10-01' });

    expect(meta.date).toBeInstanceOf(Date);
    expect(meta.tags).toEqual([]);
    expect(meta.draft).toBe(false);
  });
});

describe('sortProjects', () => {
  it('orders by explicit order, then by newest year', () => {
    const items = [
      { slug: 'c', order: 2, year: 2024 },
      { slug: 'a', order: 1, year: 2023 },
      { slug: 'b', order: 2, year: 2026 },
    ] as Project[];

    expect(sortProjects(items).map((p) => p.slug)).toEqual(['a', 'b', 'c']);
  });
});

describe('sortPosts / publishedPosts', () => {
  const posts = [
    { slug: 'old', date: new Date('2025-01-01'), draft: false },
    { slug: 'draft', date: new Date('2026-06-01'), draft: true },
    { slug: 'new', date: new Date('2026-01-01'), draft: false },
  ] as Post[];

  it('sorts newest first', () => {
    expect(sortPosts(posts).map((p) => p.slug)).toEqual(['draft', 'new', 'old']);
  });

  it('hides drafts unless explicitly included', () => {
    expect(publishedPosts(posts, { includeDrafts: false }).map((p) => p.slug)).toEqual(['new', 'old']);
    expect(publishedPosts(posts, { includeDrafts: true }).map((p) => p.slug)).toEqual(['draft', 'new', 'old']);
  });
});
