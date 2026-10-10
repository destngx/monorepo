import { dev } from '$app/environment';
import { parseCollection, publishedPosts, sortProjects } from '$lib/content/collection';
import { postMetaSchema, projectMetaSchema, type Post, type Project } from '$lib/content/schema';

const projectModules = import.meta.glob<{ metadata?: unknown }>('/src/content/projects/*.md', { eager: true });
const postModules = import.meta.glob<{ metadata?: unknown }>('/src/content/posts/*.md', { eager: true });

export const getProjects = (): Project[] => sortProjects(parseCollection(projectModules, projectMetaSchema));

export const getPosts = (): Post[] =>
  publishedPosts(parseCollection(postModules, postMetaSchema), { includeDrafts: dev });
