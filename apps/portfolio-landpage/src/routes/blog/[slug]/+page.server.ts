import { error } from '@sveltejs/kit';
import { getPosts } from '$lib/server/content';
import type { EntryGenerator, PageServerLoad } from './$types';

// getPosts() already hides drafts outside dev, so drafts are never prerendered.
export const entries: EntryGenerator = () => getPosts().map((post) => ({ slug: post.slug }));

export const load: PageServerLoad = ({ params }) => {
  const post = getPosts().find((candidate) => candidate.slug === params.slug);
  if (!post) error(404, `No post called "${params.slug}"`);
  return { post };
};
