import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'viewer/src'),
      '@components': path.resolve(__dirname, 'viewer/src/components'),
      '@ui': path.resolve(__dirname, 'viewer/src/components/ui'),
      '@lib': path.resolve(__dirname, 'viewer/src/lib'),
      '@views': path.resolve(__dirname, 'viewer/src/views'),
      '@styles': path.resolve(__dirname, 'viewer/src/styles'),
    },
  },
  test: {
    environment: 'jsdom',
    include: [
      'electron/**/*.test.ts',
      'viewer/src/**/*.test.{ts,tsx}',
    ],
    exclude: ['node_modules', 'dist', 'viewer/dist', '**/dist/**'],
    globals: true,
    setupFiles: ['./scripts/vitest.setup.ts'],
  },
});
