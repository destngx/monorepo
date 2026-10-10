import { error } from '@sveltejs/kit';
import type { Component } from 'svelte';
import { projectMetaSchema } from '$lib/content/schema';
import type { EntryGenerator, PageLoad } from './$types';

type ProjectModule = { default: Component; metadata?: unknown };

const modules = import.meta.glob<ProjectModule>('/src/content/projects/*.md');

const pathFor = (slug: string) => `/src/content/projects/${slug}.md`;

export const entries: EntryGenerator = () =>
  Object.keys(modules).map((path) => ({ slug: path.split('/').pop()!.replace(/\.md$/, '') }));

export const load: PageLoad = async ({ params }) => {
  const loadModule = modules[pathFor(params.slug)];
  if (!loadModule) error(404, `No project called "${params.slug}"`);

  const module = await loadModule();
  return { content: module.default, meta: projectMetaSchema.parse(module.metadata), slug: params.slug };
};
