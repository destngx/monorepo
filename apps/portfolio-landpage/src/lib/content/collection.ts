import type { z } from 'zod';
import type { Post, Project } from './schema';

type ContentModule = { metadata?: unknown };

const slugFromPath = (path: string): string => {
  const fileName = path.split('/').pop() ?? path;
  return fileName.replace(/\.(md|svx)$/, '');
};

/**
 * Validates the frontmatter of every module in a `import.meta.glob` result.
 * Invalid content fails the build instead of rendering a half-broken page.
 */
export function parseCollection<Schema extends z.ZodType<object>>(
  modules: Record<string, ContentModule>,
  schema: Schema,
): Array<z.infer<Schema> & { slug: string }> {
  return Object.entries(modules).map(([path, module]) => {
    const result = schema.safeParse(module.metadata);
    if (!result.success) {
      throw new Error(`Invalid frontmatter in ${path}: ${result.error.message}`);
    }
    return { ...result.data, slug: slugFromPath(path) };
  });
}

export const sortProjects = (projects: Project[]): Project[] =>
  [...projects].sort((a, b) => a.order - b.order || b.year - a.year);

export const sortPosts = (posts: Post[]): Post[] => [...posts].sort((a, b) => b.date.getTime() - a.date.getTime());

export const publishedPosts = (posts: Post[], { includeDrafts }: { includeDrafts: boolean }): Post[] =>
  sortPosts(includeDrafts ? posts : posts.filter((post) => !post.draft));
