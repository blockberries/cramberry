import { defineConfig } from 'vitest/config';
import path from 'node:path';

export default defineConfig({
  resolve: {
    alias: {
      '@cramberry/runtime': path.resolve(__dirname, '../../../typescript/src'),
    },
  },
  test: {
    globals: true,
    environment: 'node',
  },
});
