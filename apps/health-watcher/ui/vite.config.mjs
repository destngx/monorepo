import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

// The dev server proxies /api to the engine. Host and port are engine
// settings in its database now, so the dev proxy takes HW_DEV_ENGINE when the
// engine is not on the default 127.0.0.1:9191 (an IPv4 address, not
// `localhost`: Node resolves `localhost` to ::1 first).
const engine = process.env.HW_DEV_ENGINE?.trim() || 'http://127.0.0.1:9191';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit(), svelteTesting()],
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: engine,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
});
