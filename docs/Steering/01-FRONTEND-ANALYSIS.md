# Analisis Pemilihan Frontend — Claim PNC

Dokumen pendukung untuk keputusan **D-23**. Berisi perbandingan alternatif frontend modern,
dinilai terhadap batasan nyata project ini, beserta rekomendasi final dan justifikasinya.

> **Koreksi salah satu batasan (2026-09-14) — keputusannya tidak berubah.**
>
> Dokumen ini berkali-kali memakai angka **"18–27 kolom"** sebagai ukuran beratnya kebutuhan grid.
> Verifikasi terhadap export membuktikan angka itu **salah sasaran**: **median kolom sebenarnya 6**,
> dan tiga fitur grid yang biasanya paling mahal — **tambah baris inline, hapus baris inline, dan
> resize kolom** — **tidak dipakai sama sekali** di seluruh 269 section.
>
> Isi analisis di bawah **dibiarkan apa adanya** sebagai catatan pertimbangan saat keputusan
> diambil. Yang berubah adalah **konsekuensinya**, bukan pilihannya:
>
> | Hal | Sebelum | Sesudah |
> |---|---|---|
> | Pilihan framework | React + TypeScript + Vite | **tidak berubah** (`D-23`) |
> | Ukuran modul `U-2` | Besar | **Sedang** |
> | Alasan memilih pustaka tabel kelas berat | kebutuhan inti | **melemah** — kebutuhan grid lebih ringan daripada yang diperkirakan |
>
> Pilihan antara **TanStack Table** dan **AG Grid** karena itu **belum final** dan menjadi
> pertanyaan terbuka di `ADR-0002`. Masalah nyata pada grid ternyata bukan jumlah kolom melainkan
> **3.189 grid yang terikat page list klipboard** Pega dan `pyMaxRecords=500` pada 54 dari 56
> laporan — persoalan paginasi, bukan persoalan lebar tabel.

---

## 1. Batasan yang menentukan (bukan preferensi)

Pilihan frontend di sini **tidak** ditentukan oleh framework mana yang paling canggih, melainkan
oleh lima batasan keras yang sudah diputuskan:

| # | Batasan | Sumber | Konsekuensi |
|---|---|---|---|
| B1 | Tim adalah developer Pega, belum terbiasa Go/JS modern | D-09 | **Learning curve adalah faktor bobot tertinggi.** Framework dengan banyak konsep akan gagal di tangan tim ini. |
| B2 | Tampilan meniru Pega — alur dan tata letak sama | D-13 | Beban terberat ada di **grid padat dan form panjang**, bukan animasi atau halaman konten. |
| B3 | Desktop-first, tapi surveyor pakai tablet/HP | D-12 | Wajib responsive, tapi tidak perlu *mobile-first*. |
| B4 | Deploy ke VM on-premise, bukan Kubernetes | D-08 | **Menambah runtime Node.js di production adalah beban operasional nyata.** |
| B5 | Aplikasi internal di balik login, tanpa akses publik | D-07 | **SEO dan SSR tidak punya nilai sama sekali di sini.** |

Ditambah fakta terukur dari source:
- **74 layar**, **269 section**, **268 di antaranya memakai repeat/grid**
- Inbox utama punya **18–27 kolom** per tabel
- **56 laporan**, export PDF/Excel/CSV **dibuat di sisi Go** (D-11) — bukan tugas frontend
- Volume data besar (D-10) → grid wajib mendukung **paginasi server-side**, bukan memuat semua baris

> **Implikasi B5 yang sering terlewat:** karena SEO dan SSR tidak bernilai, keunggulan utama
> Next.js dan Nuxt hilang seluruhnya. Yang tersisa dari keduanya justru kerugiannya (B4).

---

## 2. Perbandingan alternatif

### React (SPA, dengan Vite)

**Kelebihan** — Ekosistem terbesar di dunia frontend. Untuk kebutuhan grid berat, pilihannya
paling matang: AG Grid dan TanGrid/TanStack Table adalah standar industri untuk tabel puluhan
kolom dengan paginasi server-side. Materi belajar berlimpah, termasuk dalam bahasa Indonesia.
Pasar tenaga kerja paling besar — paling mudah mencari pengganti bila ada developer keluar.

**Kekurangan** — Tidak opinionated. Tidak ada cara resmi untuk routing, state, form, maupun
pengambilan data; semuanya keputusan tim. Untuk tim di B1, "banyak jalan menuju Roma" berubah
menjadi **kode yang berbeda gaya di setiap layar**. Konsep yang harus dikuasai relatif banyak:
JSX, hooks, dependency array, aturan re-render, memoization. Kesalahan pada `useEffect` dan
dependency array adalah sumber bug paling umum bagi pemula.

**Performa** — Sangat baik untuk skala ini. Virtual DOM lebih lambat dibanding pendekatan
kompilasi (Svelte/Solid), tapi pada 200–300 user dengan grid ber-paginasi, perbedaannya tidak
akan terasa.

**Maintenance** — Baik bila konvensi ditegakkan; buruk bila tidak. Sangat bergantung pada
disiplin Steering.

**Kecocokan dengan Golang** — Sempurna. SPA murni, Go menyajikan JSON dan berkas statis.

**Skalabilitas** — Terbukti pada aplikasi jauh lebih besar dari ini.

**Ekosistem** — Terbaik di antara semua kandidat.

**Learning curve** — Menengah. Mudah dimulai, sulit dikuasai dengan benar.

---

### Next.js

**Kelebihan** — Kerangka lengkap di atas React: routing berbasis berkas, SSR/SSG, optimasi
gambar, server actions.

**Kekurangan** — **Seluruh keunggulan utamanya tidak relevan di sini** (B5: aplikasi internal
di balik login, tanpa SEO). Yang tersisa adalah biayanya: **butuh Node.js berjalan permanen di
VM production** (B4), model App Router dengan Server Components menambah satu lapisan konsep
berat (server vs client component, caching, streaming) yang justru paling berbahaya bagi tim B1.

**Kecocokan dengan Golang** — Buruk secara arsitektur. Kita jadi punya **dua backend runtime**
(Go dan Node) untuk satu aplikasi — dua hal yang harus di-deploy, dimonitor, ditambal
keamanannya, dan di-restart.

**Alasan tidak dipilih** — Membayar seluruh biaya SSR tanpa memperoleh satu pun manfaatnya.

---

### Vue 3 (SPA, dengan Vite)

**Kelebihan** — Learning curve paling landai di antara framework yang benar-benar matang.
Sintaks template adalah **HTML dengan direktif** (`v-if`, `v-for`, `v-model`) — secara konsep
**paling dekat dengan cara Pega Section bekerja**, sehingga tim B1 membaca kode baru dengan
model mental yang sudah mereka punya. Single-File Component menyatukan template, logika, dan
gaya dalam satu berkas yang mudah ditelusuri. Punya **pustaka resmi** untuk routing (Vue Router)
dan state (Pinia) — mengurangi keputusan yang harus dibuat tim. `v-model` menjadikan form dua
arah jauh lebih ringkas dibanding React, dan **B2 adalah aplikasi yang didominasi form**.

**Kekurangan** — Ekosistem lebih kecil dari React. Pasar tenaga kerja di Indonesia lebih kecil
walau tetap sehat. Beberapa pustaka niche hanya tersedia untuk React.

**Performa** — Setara React, pada beberapa kasus sedikit lebih baik. Lebih dari cukup.

**Maintenance** — Sangat baik. Struktur SFC memaksa keseragaman, dan pustaka resmi mencegah
perpecahan gaya antar developer.

**Kecocokan dengan Golang** — Sempurna. SPA murni.

**Skalabilitas** — Terbukti pada aplikasi enterprise berskala besar.

**Ekosistem** — Cukup. Untuk kebutuhan spesifik kita, **PrimeVue** menyediakan DataTable dengan
paginasi server-side, filter per kolom, kolom beku, kolom yang bisa diatur ulang, seleksi baris,
dan penyuntingan sel — yaitu **persis daftar kebutuhan grid Pega**. Element Plus dan Naive UI
adalah alternatif setara.

**Learning curve** — Paling landai. Ini keunggulan terbesarnya untuk project ini.

---

### Nuxt

**Kelebihan** — Setara Next.js untuk ekosistem Vue.

**Kekurangan** — Persis sama dengan Next.js: manfaat SSR tidak terpakai (B5), tapi kewajiban
menjalankan Node.js di VM tetap ada (B4).

**Alasan tidak dipilih** — Sama dengan Next.js.

---

### Angular

**Kelebihan** — Paling opinionated dari semua kandidat, dan itu **sebenarnya cocok** dengan
kebutuhan B1 akan struktur yang seragam. Semuanya resmi dan satu jalan: routing, HTTP client,
form (reactive forms sangat kuat untuk form panjang seperti milik kita), validasi, testing,
dependency injection. TypeScript wajib sejak awal. Terbukti pada aplikasi *line-of-business*
besar di lingkungan korporat. Rilis punya jadwal panjang yang jelas.

**Kekurangan** — **Learning curve paling curam.** Tim B1 harus menyerap Dependency Injection,
dekorator, modul (atau standalone component), dan terutama **RxJS** — pemrograman reaktif
berbasis stream yang merupakan hambatan besar bagi developer yang belum pernah menyentuhnya.
Kode jauh lebih panjang untuk hasil yang sama. Ukuran bundel paling besar.

**Performa** — Baik, tapi paling berat di antara kandidat.

**Maintenance** — Sangat baik dalam jangka panjang, **jika** tim berhasil melewati fase belajar.

**Kecocokan dengan Golang** — Sempurna. SPA murni.

**Alasan tidak dipilih** — Kekuatannya (struktur ketat) bisa kita peroleh dari Steering yang
preskriptif, sedangkan kelemahannya (RxJS + DI + verbositas) langsung menghantam batasan
terkuat kita, yaitu B1. Risiko tim tidak pernah benar-benar produktif terlalu besar.

---

### Svelte / SvelteKit

**Kelebihan** — Model mental paling sederhana dari semuanya; sintaks paling sedikit boilerplate.
Dikompilasi, tanpa Virtual DOM — bundel terkecil dan performa runtime terbaik.

**Kekurangan** — Ekosistem paling kecil di antara kandidat arus utama. **Pilihan pustaka
DataTable enterprise sangat terbatas** — padahal itu kebutuhan inti kita (18–27 kolom, paginasi
server-side, filter per kolom). Pasar tenaga kerja di Indonesia sempit. SvelteKit juga membawa
persoalan runtime Node yang sama seperti Next/Nuxt bila dipakai penuh.

**Alasan tidak dipilih** — Kesederhanaannya menarik untuk B1, tapi kami akan **membangun sendiri
komponen grid** yang di ekosistem lain sudah tersedia matang. Untuk 74 layar dengan tim yang
masih belajar, itu risiko yang tidak sepadan.

---

### SolidJS

**Kelebihan** — Performa terbaik secara benchmark. Reaktivitas granular tanpa Virtual DOM.
Sintaks mirip React sehingga materi React sebagian bisa dipakai.

**Kekurangan** — Ekosistem paling kecil dan komunitas paling sedikit. Nyaris tidak ada pustaka
komponen enterprise yang matang. Pasar tenaga kerja sangat sempit.

**Alasan tidak dipilih** — Terlalu berisiko untuk aplikasi bisnis inti berumur panjang yang
dikerjakan tim yang sedang belajar. Keunggulan performanya tidak menjawab masalah nyata kita —
hambatan kita adalah volume data di sisi database, bukan kecepatan render.

---

### Alternatif lain: Go + `templ` + HTMX (server-rendered)

Layak dipertimbangkan serius, bukan sekadar pelengkap daftar.

**Kelebihan** — **Satu bahasa untuk seluruh aplikasi**, dan itu menjawab B1 secara paling
langsung: tim hanya perlu belajar Go, bukan Go *dan* satu ekosistem JavaScript. Satu artefak
deploy, satu proses yang dimonitor — sangat cocok dengan B4. Secara konseptual **paling dekat
dengan Pega**, yang juga merender HTML di sisi server. Tanpa proses build frontend, tanpa
`node_modules`.

**Kekurangan** — Titik lemahnya persis di tempat aplikasi kita paling berat. Grid dengan
penyuntingan inline, baris yang bisa ditambah/dihapus, dan **perhitungan langsung di layar**
(total spreading wajib 100%, nilai settlement yang berubah seketika) memerlukan interaksi sisi
klien yang kaya. Dengan HTMX, setiap interaksi kecil menjadi perjalanan bolak-balik ke server —
pada form panjang milik Pega ini akan terasa lambat dan rapuh, terutama untuk surveyor di
jaringan seluler (B3). Menambal ini dengan Alpine.js berarti tetap menulis JavaScript, hanya
dengan alat yang jauh lebih terbatas.

**Alasan tidak dipilih** — Tepat menyelesaikan masalah tim, tapi tepat gagal pada karakter UI
yang harus kita tiru (B2). Bila UI boleh disederhanakan, opsi ini akan menjadi rekomendasi
utama — namun D-13 menutup kemungkinan itu.

---

## 3. Tabel ringkas

Bobot mencerminkan batasan project, bukan mutu framework secara umum.

| Kriteria | Bobot | React | Next.js | **Vue 3** | Nuxt | Angular | Svelte | SolidJS | Go+HTMX |
|---|---|---|---|---|---|---|---|---|---|
| Learning curve untuk tim eks-Pega | ★★★★★ | 3 | 2 | **5** | 2 | 1 | 4 | 3 | 5 |
| Kesiapan grid/tabel enterprise | ★★★★★ | 5 | 5 | **5** | 5 | 5 | 2 | 1 | 2 |
| Kemudahan form panjang & kompleks | ★★★★☆ | 3 | 3 | **5** | 5 | 5 | 4 | 3 | 2 |
| Kecocokan dengan backend Go | ★★★★☆ | 5 | 2 | **5** | 2 | 5 | 5 | 5 | 5 |
| Kesederhanaan operasional di VM | ★★★★☆ | 5 | 1 | **5** | 1 | 5 | 5 | 5 | 5 |
| Keseragaman kode / anti-perpecahan gaya | ★★★★☆ | 2 | 3 | **4** | 4 | 5 | 3 | 2 | 4 |
| Ekosistem & pustaka | ★★★☆☆ | 5 | 5 | **4** | 4 | 4 | 2 | 1 | 3 |
| Ketersediaan SDM di Indonesia | ★★★☆☆ | 5 | 5 | **4** | 3 | 3 | 2 | 1 | 3 |
| Performa untuk 200–300 user | ★★☆☆☆ | 4 | 4 | **4** | 4 | 3 | 5 | 5 | 4 |
| Nilai SSR/SEO | — | — | 0 | — | 0 | — | — | — | — |
| Responsif untuk surveyor (B3) | ★★★☆☆ | 5 | 5 | **5** | 5 | 4 | 5 | 5 | 3 |

---

## 4. Rekomendasi final

### **Vue 3 + TypeScript + Vite + PrimeVue**, sebagai SPA murni yang disajikan oleh binary Go.

**Justifikasi teknis:**

1. **Menjawab batasan terkuat secara langsung (B1).** Learning curve paling landai di antara
   opsi yang matang. Sintaks template Vue adalah HTML dengan direktif — model mental yang sudah
   dimiliki tim dari Pega Section. Ini menghemat berbulan-bulan produktivitas dibanding React
   (hooks, re-render) atau Angular (RxJS, DI).

2. **Kuat tepat di titik terberat aplikasi (B2).** Aplikasi ini didominasi form panjang dan grid
   padat. `v-model` membuat form dua arah jauh lebih ringkas dan lebih sulit disalahgunakan
   dibanding pola controlled component React. PrimeVue DataTable menyediakan paginasi
   server-side, filter per kolom, kolom beku, dan penyuntingan sel — persis kebutuhan grid
   18–27 kolom kita, tanpa membangun sendiri.

3. **Mengurangi keputusan yang harus dibuat tim.** Vue Router dan Pinia adalah pustaka resmi.
   Tim tidak perlu memilih di antara lima pustaka routing dan tujuh pustaka state. Untuk tim
   yang sedang belajar, **lebih sedikit keputusan berarti lebih sedikit ketidakseragaman** —
   dan ini melengkapi D-09 yang menuntut Steering preskriptif.

4. **Menjaga kesederhanaan operasional (B4, B8).** SPA murni dikompilasi menjadi berkas statis
   yang bisa **disajikan langsung oleh binary Go**. Di VM production hanya ada **satu proses**
   untuk di-deploy dan dimonitor. Next.js dan Nuxt akan menambah runtime Node.js kedua tanpa
   memberi manfaat apa pun (B5).

5. **TypeScript wajib, bukan opsional.** Pada 74 layar yang dikerjakan tim yang sedang belajar,
   TypeScript menangkap kesalahan saat kompilasi yang jika tidak akan lolos ke production.
   Tambahan beban belajarnya nyata namun terbayar berkali lipat, dan tipe dapat **dihasilkan
   otomatis dari kontrak API Go** sehingga backend dan frontend tidak pernah berbeda persepsi.

**Alternatif kedua: React + TypeScript + Vite + TanStack Table / AG Grid.**
Dipilih **jika** faktor penentu bergeser ke ketersediaan SDM jangka panjang. Ekosistem dan
pasar kerja React lebih besar, dan itu keunggulan nyata untuk aplikasi berumur 10 tahun. Yang
dikorbankan: learning curve lebih curam dan risiko ketidakseragaman kode lebih tinggi — yang
harus ditebus dengan Coding Standards dan code review yang jauh lebih ketat.

**Yang secara tegas tidak direkomendasikan:** Next.js dan Nuxt (membayar biaya SSR tanpa
manfaatnya, serta menambah runtime kedua di VM), Svelte dan SolidJS (ekosistem grid enterprise
belum memadai untuk kebutuhan inti kita).
