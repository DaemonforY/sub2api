import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { readFileSync } from 'node:fs'

// Served by the Go binary on video.<domain> (see backend/internal/web/embed_on.go serveVideo).
// Dev: API and gateway calls go to VITE_API_TARGET (a local backend or the mock).
export default defineConfig({
  plugins: [
    vue(),
    {
      // The Go server reads the tool landing pages for their titles and the sitemap.
      name: 'seo-tools',
      generateBundle() {
        this.emitFile({ type: 'asset', fileName: 'seo-tools.json', source: readFileSync(new URL('./src/lib/tools.json', import.meta.url)) })
      }
    }
  ],
  server: {
    proxy: {
      '/api': { target: process.env.VITE_API_TARGET || 'http://localhost:8080', changeOrigin: true },
      '/v1': { target: process.env.VITE_API_TARGET || 'http://localhost:8080', changeOrigin: true }
    }
  },
  build: { outDir: 'dist', assetsDir: 'assets', chunkSizeWarningLimit: 900 }
})
