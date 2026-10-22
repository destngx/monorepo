import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  // Outside a SvelteKit app, $app/navigation is a stub that tests mock.
  resolve: {
    alias: { '$app/navigation': fileURLToPath(new URL('./src/test/app-navigation.ts', import.meta.url)) },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    // Node 25+ ships its own localStorage global, which shadows jsdom's.
    execArgv: ['--no-experimental-webstorage'],
  },
});
