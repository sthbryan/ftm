import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
  plugins: [preact(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // Sephiro ships React components; Preact's compat layer serves both.
      react: 'preact/compat',
      'react-dom': 'preact/compat',
      'react/jsx-runtime': 'preact/jsx-runtime',
    },
  },
  server: {
    // `FTM_API=http://127.0.0.1:<port> bun run dev` talks to a running `ftm --server`.
    proxy: process.env.FTM_API ? { '/api': process.env.FTM_API } : undefined,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
