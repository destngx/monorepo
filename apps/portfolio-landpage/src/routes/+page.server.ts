import { getPosts, getProjects } from '$lib/server/content';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = () => ({
  featured: getProjects().filter((project) => project.featured),
  projectCount: getProjects().length,
  posts: getPosts().slice(0, 3),
});
