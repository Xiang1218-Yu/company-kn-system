import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Vite config. The dev server proxies /api to the Go backend so the frontend
// runs on :5173 and shares cookies/origins with the API during development.
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': {
        // Default to :8088 (the docker-compose host port). The host's :8080 is
        // taken by another local container (express-manage), so pointing the
        // dev proxy at :8088 avoids the 405 from that unrelated service. Override
        // with VITE_API_URL if you run the backend elsewhere.
        target: process.env.VITE_API_URL || 'http://localhost:8088',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})
