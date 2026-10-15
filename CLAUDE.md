# claim-pnc-frontend — aturan kerja

Frontend Claim PNC: SPA statis React 19 + TypeScript 7 + Vite 8, Tailwind v4, TanStack Query, react-hook-form + zod, react-router 7. Hasil build disematkan ke binary Go backend — tidak ada runtime Node.js di produksi (ADR-0002). Backend ada di `../claim-pnc-backend`; Steering, ADR, Decision Log, dan tiket di `../claim-pnc-context/docs/` (istilah domain: `docs/Steering/CONTEXT.md`).

## Menjalankan

- Node 26 (`.nvmrc`, `engines`). `npm ci`, lalu `npm run dev` → http://localhost:5173. Panggilan `/api` di-proxy ke backend di `http://localhost:8080` — jalankan backend dulu (`go run ./cmd/claimpnc` di `../claim-pnc-backend`).
- `vite.config.ts` dan script `mark-dist` menulis build ke `../backend/spa/dist` (tata letak monorepo lama). Di workspace ini backend bernama `claim-pnc-backend`, jadi untuk menyematkan ke binary: `npx vite build --outDir ../claim-pnc-backend/spa/dist --emptyOutDir`, lalu pastikan `spa/dist/.gitkeep` di backend tetap ada (`emptyOutDir` menghapusnya).

## Struktur

```
src/api/            client.ts (callAPI, uploadAPI, downloadAPI, APIError, NetworkError), types.ts
src/app/            App.tsx (seluruh <Route>), SessionGuard, portal.ts, session.ts, Sidebar
src/app/menu/       registry.ts — peta MENU_PROGRAM → rute layar
src/components/     komponen bersama (Field, DataTable, DateField, NumberField, Icon, …) + format.ts
src/lib/            logika murni bersama (money, claim, sort)
src/modules/<modul>/  api.ts (hook TanStack Query), <Nama>Page.tsx, <Nama>Form.tsx, *.test.tsx
```

Nama folder modul kebab-case dan mengikuti modul backend (`master-status-klaim` ↔ `internal/masterstatus`).

## Aturan yang mengikat

- **Tidak ada logika bisnis di frontend.** Validasi hanya cermin aturan backend; backend yang memutuskan. `SessionGuard` adalah kenyamanan tampilan — penegakan sesi dan hak akses ada di server.
- **Menambah layar = dua tempat:** `<Route>` di `src/app/App.tsx` (dibungkus `SessionGuard`) dan satu baris di `src/app/menu/registry.ts`. Kunci registry harus sama persis dengan `MENU_PROGRAM` di `POOLDATA.M_MENU_APLIKASI_PNC`, termasuk huruf besar-kecil — salah ketik tidak menimbulkan error, menunya hanya tetap "belum tersedia".
- **Rute API ditulis lengkap** `'/api/<rute>'` (tanpa `/v1`) dan harus sama persis dengan `Mount` di backend. Panggilan API hanya di `api.ts` modul, lewat `src/api/client.ts`: `callAPI` untuk JSON, `uploadAPI` untuk unggah, `downloadAPI`/`blobAPI` untuk ekspor & unduhan. (Beberapa modul inbox masih memakai `fetch` langsung untuk ekspor; jangan ditiru.)
- **Portal masuk ke kunci cache.** Setiap query key data entitas memuat portal aktif (`useSelectedPortal`) — tanpa itu, berpindah portal menampilkan data badan hukum lain dari cache (R-20).
- **Galat dibedakan lewat `APIError.kode`, bukan teks `pesan`.** Galat per isian dibaca lewat `err.violations()` (menyatukan bentuk `detail` dan `field` yang belum seragam di backend) dan ditandai di medannya.
- **Uang dikirim sebagai sen (bilangan bulat), persen sebagai ×10.000** (ADR-0016). Jangan menghitung dengan pecahan; format tampilan hanya lewat `src/components/format.ts`.
- **Konfirmasi & pop-up di dalam halaman, bukan `window.confirm`/`alert`** — dialog peramban tidak dapat diuji maupun diberi gaya. (Tiga layar konversi masih memakai `window.confirm`; jangan ditiru.)
- **Tanpa dependensi dari internet saat runtime:** aplikasi berjalan di VM on-premise jaringan tertutup. Ikon digambar sendiri di `src/components/Icon.tsx` (bukan pustaka ikon), font sistem (bukan Google Fonts). Dependensi baru perlu alasan kuat.
- **Peramban tertua: Chrome 109 (Windows 7).** Jangan memakai API JS/CSS yang lebih baru tanpa fallback; warna Tailwind `oklch()` diturunkan Lightning CSS — jangan mematikan pengaturan `css.lightningcss`/`build.target` di `vite.config.ts`.
- **Tema:** `src/styles.css` (Glassmorphism Navy sejak 2026-09-28, ditetapkan Work Owner). `src/gaya.css` adalah tema terang lama dan tidak di-import. Pakai token/kelas yang ada, bukan warna mentah.
- Teks tampilan Bahasa Indonesia, mengikuti label layar Pega lama dan istilah `CONTEXT.md` — tidak diterjemahkan atau "dirapikan".
- Setiap aturan yang dipindahkan dari Pega menyebut sumbernya di komentar (rule/harness Pega, tabel Oracle, atau nomor D-/ADR-/R-/TKT-).

## Gerbang sebelum menyatakan selesai

```
npm run typecheck && npm test && npm run build
```

Setiap perubahan kode disertai test untuk kode yang disentuh (Vitest + Testing Library, `*.test.tsx` di samping berkasnya). CI GitLab juga menjalankan lint dan `npm run test:coverage`.

- Seluruh suite ~8 menit; selama mengerjakan, jalankan test yang relevan saja: `npx vitest run src/modules/<modul>`.
- Test otomatis berjalan di zona waktu `Asia/Jakarta` (diatur di `vite.config.ts`) — tulis ekspektasi tanggal dalam WIB.
- Keluaran `Not implemented: navigation to another Document` dari jsdom adalah derau, bukan kegagalan.
- Lint: `npm run lint:install` sekali (memasang ESLint + TypeScript 5.9 terpisah di `tools/lint/`, karena typescript-eslint belum mendukung TS 7), lalu `npm run lint`.
- `npm run build` dan `npm run dev` tidak memeriksa tipe; `tsc` pun tidak menangkap ketidakcocokan nama medan JSON atau rute dengan backend. Setelah mengubah kontrak, uji layar terhadap backend yang dijalankan ulang.
