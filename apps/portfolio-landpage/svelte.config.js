import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { mdsvex } from 'mdsvex';
import { highlightCode } from './src/lib/content/highlight.js';

const OPTIONAL_DYNAMIC_ROUTES = new Set(['/blog/[slug]']);

/** @type {import('@sveltejs/kit').Config} */
const config = {
  extensions: ['.svelte', '.md'],
  preprocess: [
    vitePreprocess(),
    mdsvex({
      extensions: ['.md'],
      highlight: { highlighter: highlightCode },
    }),
  ],
  compilerOptions: {
    // Rune mode for project files only; node_modules decide for themselves.
    runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true),
    // mdsvex still emits <script context="module">; the deprecation is not actionable for .md files.
    warningFilter: (warning) => !(warning.code === 'script_context_deprecated' && warning.filename?.endsWith('.md')),
  },
  kit: {
    // Static hosts serve 404.html for unknown paths; it boots the app, which renders +error.svelte.
    adapter: adapter({ strict: true, fallback: '404.html' }),
    prerender: {
      handleHttpError: 'fail',
      handleMissingId: 'fail',
      // Dynamic routes may legitimately be empty (e.g. no published posts yet); anything else is a bug.
      handleUnseenRoutes: ({ routes, message }) => {
        const unexpected = routes.filter((route) => !OPTIONAL_DYNAMIC_ROUTES.has(route));
        if (unexpected.length) throw new Error(message);
      },
    },
  },
};

export default config;
