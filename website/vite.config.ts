import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// In production Traefik serves the SPA and the API from one origin, so the app
// uses relative URLs. Mirror that in dev by proxying /api and /health to a real
// API server instead of setting VITE_API_BASE_URL, which keeps dev and prod on
// the same code path. Override the target with VITE_DEV_API_TARGET.
const devApiTarget = process.env.VITE_DEV_API_TARGET || 'http://bbb1.wegman:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': { target: devApiTarget, changeOrigin: true, ws: true },
      '/health': { target: devApiTarget, changeOrigin: true },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/setupTests.ts',
    css: true,
  },
})
