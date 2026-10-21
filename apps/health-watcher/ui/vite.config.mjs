import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

// The engine listens on HOST (default 127.0.0.1, IPv4 only). Proxy to that
// address, not `localhost`: Node resolves `localhost` to ::1 first.
const engineHost = (() => {
  const host = process.env.HOST?.trim();
  if (!host || host === '0.0.0.0' || host === 'localhost') return '127.0.0.1';
  if (host === '::') return '[::1]';
  return host.includes(':') ? `[${host}]` : host;
})();

export default defineConfig({
  plugins: [tailwindcss(), sveltekit(), svelteTesting()],
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: `http://${engineHost}:${process.env.PORT?.trim() || 9191}`,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
});
