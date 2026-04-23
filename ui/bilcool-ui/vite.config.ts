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
      '/api/v1/users': { target: process.env.AUTH_SERVICE_URL ?? 'http://localhost:8082' },
      '/api/v1/bookings': { target: process.env.BOOK_SERVICE_URL ?? 'http://localhost:8081' },
      '/api/v1/events': { target: process.env.EVENTS_SERVICE_URL ?? 'http://localhost:8083' },
      '/api/v1/journal': { target: process.env.JOURNAL_SERVICE_URL ?? 'http://localhost:8084' },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
});
