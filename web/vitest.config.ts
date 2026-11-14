import { defineConfig } from 'vitest/config'

// Unit tests for the rendering pipeline. jsdom gives DOMPurify a DOM, so
// tests run the same sanitisation as the browser.
export default defineConfig({
  test: { environment: 'jsdom', include: ['src/**/*.test.ts'] },
})
