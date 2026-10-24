import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The Go server embeds dist/. No inline scripts are emitted, so the strict
// CSP (script-src 'self') holds.
export default defineConfig({
  plugins: [svelte()],
  css: { target: 'chrome87' },
  build: {
    outDir: 'dist',
    // Older Android browsers (e.g. Brave on LineageOS) must still parse the bundle.
    target: ['es2020', 'chrome87', 'safari14', 'firefox78'],
    emptyOutDir: true,
    assetsInlineLimit: 0,
    modulePreload: { polyfill: false },
    chunkSizeWarningLimit: 1200,
  },
})
