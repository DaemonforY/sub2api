import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// Served at https://hivegpt.cn/editor/ under a strict CSP:
// script-src 'self' (no 'unsafe-eval', no inline scripts). The SFC template is
// precompiled, the entry is an external module script, and the modulepreload
// polyfill (which Vite would otherwise inline) is turned off.
// `vite preview` mimics the production CSP so CSP regressions show up locally.
const PREVIEW_CSP = [
  "default-src 'self'",
  "script-src 'self' 'nonce-preview'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https:",
  "connect-src 'self'",
  "font-src 'self' data:",
  "object-src 'none'",
  "base-uri 'self'",
  "frame-ancestors 'none'"
].join('; ');

export default defineConfig({
  base: '/editor/',
  plugins: [vue()],
  server: {
    // 本地开发：/api 转发到主站后端（或 canvas-mock/main-mock.py 的假接口，端口 8092）
    proxy: {
      '/api': { target: process.env.EDITOR_API_PROXY || 'http://localhost:8092', changeOrigin: true }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    modulePreload: { polyfill: false },
    assetsInlineLimit: 0
  },
  preview: {
    headers: { 'Content-Security-Policy': PREVIEW_CSP },
    proxy: {
      '/api': { target: process.env.EDITOR_API_PROXY || 'http://localhost:8092', changeOrigin: true }
    }
  }
});
