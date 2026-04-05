/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api/auth': { target: 'http://localhost:8082', rewrite: (path) => path.replace('/api/auth', '') },
      '/api/book': { target: 'http://localhost:8081', rewrite: (path) => path.replace('/api/book', '') },
      '/api/events': { target: 'http://localhost:8083', rewrite: (path) => path.replace('/api/events', '') },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
});
