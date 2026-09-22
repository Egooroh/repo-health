import { fileURLToPath, URL } from 'node:url'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: fileURLToPath(new URL('../dist', import.meta.url)),
    emptyOutDir: true,
  },
  server: {
    proxy: {
      // changeOrigin: false keeps the original Host header (localhost:5173)
      // so it still matches the browser's Origin header — the backend's
      // POST middleware 403s otherwise, since it compares the two.
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: false },
      '/healthz': { target: 'http://127.0.0.1:8080', changeOrigin: false },
    },
  },
})
