import adapter from '@sveltejs/adapter-static';
import { relative, sep } from 'node:path';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    // defaults to rune mode for the project, except for `node_modules`. Can be removed in svelte 6.
    runes: ({ filename }) => {
      const relativePath = relative(import.meta.dirname, filename);
      const pathSegments = relativePath.toLowerCase().split(sep);
      const isExternalLibrary = pathSegments.includes('node_modules');

      return isExternalLibrary ? undefined : true;
    },
  },
  kit: {
    // Built as an SPA and served by the FastAPI app (see apps/graph-weave/src/ui.py).
    adapter: adapter({ fallback: 'index.html' }),
    paths: { base: '/ui' },
  },
};

export default config;
