import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Tauri loads the dev server on a fixed port and the built files from dist/.
export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: { port: 1420, strictPort: true },
  envPrefix: ['VITE_', 'TAURI_ENV_'],
  build: { target: 'es2021', outDir: 'dist' }
})
