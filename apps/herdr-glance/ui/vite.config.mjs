import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

// The dev server proxies /api to the engine (HERDR_GLANCE_DEV_ENGINE, default 127.0.0.1:9393).
const engine = process.env.HERDR_GLANCE_DEV_ENGINE?.trim() || 'http://127.0.0.1:9393';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit(), svelteTesting()],
  server: {
    port: 5176,
    proxy: {
      '/api': {
        target: engine,
        // Keep the browser's Host and Origin, so the engine's same-origin check passes in dev.
        changeOrigin: false,
      },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
});
