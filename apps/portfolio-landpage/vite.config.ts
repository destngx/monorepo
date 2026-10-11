import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit(), svelteTesting()],
  build: {
    // The largest chunk is three.js (about 810 kB), loaded lazily by Hero.svelte; anything bigger is a regression.
    chunkSizeWarningLimit: 850,
  },
  test: {
    environment: 'jsdom',
    environmentOptions: { jsdom: { url: 'http://localhost/' } },
    include: ['src/**/*.{test,spec}.ts'],
    setupFiles: ['./vitest-setup.ts'],
    // Node 25+ ships its own localStorage global, which shadows jsdom's.
    execArgv: ['--no-experimental-webstorage'],
  },
});
