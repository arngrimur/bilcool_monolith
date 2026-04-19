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
      '/api/auth': { target: process.env.AUTH_SERVICE_URL ?? 'http://localhost:8082', rewrite: (path) => path.replace('/api/auth', '') },
      '/api/book': { target: process.env.BOOK_SERVICE_URL ?? 'http://localhost:8081', rewrite: (path) => path.replace('/api/book', '') },
      '/api/events': { target: process.env.EVENTS_SERVICE_URL ?? 'http://localhost:8083', rewrite: (path) => path.replace('/api/events', '') },
      '/api/journal': { target: process.env.JOURNAL_SERVICE_URL ?? 'http://localhost:8084', rewrite: (path) => path.replace('/api/journal', '') },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
});
