import { error } from '@sveltejs/kit';
import type { Component } from 'svelte';
import type { PageLoad } from './$types';

const modules = import.meta.glob<{ default: Component }>('/src/content/posts/*.md');

export const load: PageLoad = async ({ data, params }) => {
  const loadModule = modules[`/src/content/posts/${params.slug}.md`];
  if (!loadModule) error(404, `No post called "${params.slug}"`);

  return { ...data, content: (await loadModule()).default };
};
