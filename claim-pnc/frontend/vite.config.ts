import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Hasil build masuk ke backend/spa/dist, lalu disematkan ke binary Go (ADR-0002).
// Tidak ada runtime Node.js di produksi.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: '../backend/spa/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    // Selama pengembangan, panggilan /api diteruskan ke binary Go yang berjalan
    // terpisah. Di produksi keduanya dilayani proses yang sama sehingga proxy ini
    // tidak ikut ter-build.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
