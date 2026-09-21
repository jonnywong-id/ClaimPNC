import { fileURLToPath, URL } from 'node:url'

import { createLogger } from 'vite'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Alamat binary Go saat pengembangan. Disebut sekali supaya pesan kesalahan dan sasaran
// proxy tidak pernah menyebut alamat yang berbeda.
const BACKEND = 'http://localhost:8080'

// Ditegakkan penangan galat proxy di bawah tepat sebelum Vite mencetak pesan bawaannya.
//
// Vite memasang penangan 'error'-nya SENDIRI setelah `configure` dipanggil, dan penangan
// itu tidak dapat dilepas dari sini. Yang dapat dilakukan adalah menyaring keluarannya —
// dan hanya keluaran yang sudah kita jelaskan sendiri, supaya galat proxy yang benar-benar
// urusan proxy tetap tercetak utuh.
let sudahDijelaskan = false

const logger = createLogger()
const cetakGalat = logger.error.bind(logger)
logger.error = (pesan, opsi) => {
  if (sudahDijelaskan && typeof pesan === 'string' && pesan.includes('http proxy error')) {
    sudahDijelaskan = false
    return
  }
  cetakGalat(pesan, opsi)
}

// Hasil build masuk ke backend/spa/dist, lalu disematkan ke binary Go (ADR-0002).
// Tidak ada runtime Node.js di produksi.
export default defineConfig({
  customLogger: logger,
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
      '/api': {
        target: BACKEND,

        // Ketika backend TIDAK menyala, Vite secara bawaan mencetak
        // `http proxy error: /api/masuk` — kalimat yang menyalahkan proxy padahal
        // proxy-nya baik-baik saja; yang tidak ada adalah lawan bicaranya. Pesan itu
        // sudah dua kali membuat waktu terbuang menelusuri tempat yang salah.
        //
        // Penanganan di bawah memisahkan dua hal yang bawaannya tercampur:
        //
        //   sambungan ditolak/putus  -> backend belum menyala. Soket DIPUTUS, sehingga
        //                               `fetch` di peramban gagal dan layar menampilkan
        //                               "Server Claim PNC tidak dapat dihubungi" —
        //                               kalimat yang memang sudah ditulis untuk keadaan
        //                               ini, bukan galat proxy.
        //   galat lain               -> benar-benar urusan proxy. Dicetak apa adanya
        //                               beserta kodenya, karena di situlah pesan
        //                               "proxy error" memang pada tempatnya.
        configure: (proxy) => {
          proxy.on('error', (err: NodeJS.ErrnoException, _req, res) => {
            const backendDown =
              err.code === 'ECONNREFUSED' ||
              err.code === 'ECONNRESET' ||
              err.code === 'EHOSTUNREACH' ||
              err.code === 'ETIMEDOUT'

            if (backendDown) {
              console.error(
                `\n  Backend Claim PNC tidak menyala di ${BACKEND}.\n` +
                  '  Jalankan di terminal lain:  cd claim-pnc/backend && go run ./cmd/claimpnc\n' +
                  '  (Ini BUKAN galat proxy — proxy-nya berjalan, lawan bicaranya yang tidak ada.)\n',
              )
              // Pesan bawaan Vite untuk galat INI saja yang ditelan; yang berikutnya
              // dicetak utuh lagi.
              sudahDijelaskan = true
            } else {
              console.error(`\n  Galat proxy ke ${BACKEND}: ${err.code ?? ''} ${err.message}\n`)
            }

            // `res` dapat berupa ServerResponse (HTTP biasa) atau Socket (upgrade
            // WebSocket). Keduanya punya cara diputus yang berbeda, dan memanggil yang
            // salah melempar galat kedua yang menutupi galat aslinya.
            if ('destroy' in res && typeof res.destroy === 'function' && !('socket' in res)) {
              res.destroy()
              return
            }
            if ('socket' in res) res.socket?.destroy()
          })
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
