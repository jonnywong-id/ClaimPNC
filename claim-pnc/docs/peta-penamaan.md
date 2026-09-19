# Peta Penamaan — Indonesia → Inggris

Rujukan untuk `D-80` (nama di dalam kode berbahasa Inggris) dan `D-81` (nama folder modul
memakai nama modul bisnis). Dokumen ini menjawab dua pertanyaan: **apa yang berubah**, dan
**apa yang sengaja tidak berubah**.

| | |
|---|---|
| Tanggal | 2026-09-18 |
| Dasar | `D-80`, `D-81`, `08-TECHNICAL-STRATEGY.md` §4.1 |
| Cakupan | seluruh `claim-pnc/backend` dan `claim-pnc/frontend/src` |
| Bukti | backend: `go build`/`go vet`/`go test` bersih (10 paket) · frontend: `tsc --noEmit` bersih, 33 uji lulus, `npm run build` berhasil |

---

## 1. Aturannya dalam satu tabel

| Hal | Bahasa | Alasan |
|---|---|---|
| Folder, berkas, paket, tipe, fungsi, method, field, parameter, variabel | **Inggris** | Pustaka standar Go dan React seluruhnya Inggris; penamaan campur membuat satu baris berpindah bahasa dua kali |
| **Nama folder modul** | **Indonesia** | Nama yang dipakai Work Owner saat meminta pekerjaan dan yang tertulis di tiket (`D-81`) |
| Komentar dan seluruh dokumen di `docs/` | **Indonesia** | Pembacanya tim, dan `D-09` menetapkan tim adalah developer Pega internal |
| Nama field JSON pada API | **Indonesia** | Ia **kontrak**. Mengubahnya merusak klien, bukan mengganti nama |
| Nama tabel dan kolom basis data | **Indonesia** | Dimiliki bersama Pega selama masa paralel (`D-21`); perubahannya menempuh `D-63` |
| Jalur URL API (`/api/masuk`, `/api/registrasi/klaim`) | **Indonesia** | Sama sifatnya dengan nama field JSON |
| Teks yang dilihat pengguna | **Indonesia** | Mengikuti layar Pega apa adanya (`D-13`) |
| Nama variabel lingkungan dan flag baris perintah | **Indonesia** | Dipakai berkas `.env`, skrip deployment, dan operator |
| Nama kueri di berkas `.sql` (`-- name: klaim_sisip`) | **Indonesia** | Ia hidup berdampingan dengan nama tabel dan kolom di berkas yang sama |

Akibatnya satu baris dapat memuat keduanya, dan itu memang yang dikehendaki:

```go
// Number adalah nomor rekening; namanya di basis data tetap NO_REKENING.
type Account struct {
	Number string `json:"nomor_rekening"`
}
```

### Yang TIDAK berubah dari `D-19`

`D-19` menetapkan istilah domain tidak boleh memakai alias Pega yang salah arti. Sasaran itu
**tetap dipegang**; yang berubah hanya bahasanya. Padanan yang dipakai adalah padanan benar dari
`CONTEXT.md`, bukan nama lama Pega:

| Alias Pega | **Tidak** dipakai | Dipakai |
|---|---|---|
| `Adjustment` | ❌ | `SettlementLine` |
| `Object` | ❌ | `InsuredItem` |
| `CaseID` | ❌ | nama eksplisit per konteks |

---

## 2. Folder yang berganti nama

| Sebelum | Sesudah | Catatan |
|---|---|---|
| `backend/internal/platform/waktu` | `backend/internal/platform/clock` | sekaligus menyamakan nama dengan seam Clock di Steering |
| `backend/internal/*/repo/memori` | `backend/internal/*/repo/memory` | tiga modul |
| `frontend/src/modules/masuk` | `frontend/src/modules/login` | nama modulnya memang "Login" |
| `frontend/src/modules/beranda` | `frontend/src/modules/home` | nama modulnya memang "Home" |
| `frontend/src/uji` | `frontend/src/test` | |

**Yang sengaja tetap:** `backend/internal/registrasi`, `frontend/src/modules/registrasi`,
`frontend/src/modules/portal`, `backend/internal/portal`, `backend/internal/auth`,
`backend/internal/platform`, `backend/spa`. Empat yang terakhir adalah modul kerangka yang tidak
punya nama bisnis; dua yang pertama memakai nama modul (`D-81`).

Bentuk nama folder modul berbeda antar lapisan karena Go melarang tanda hubung pada nama paket:

| Lapisan | Bentuk | Contoh |
|---|---|---|
| Backend — folder dan nama paket Go | `namamodul`, tanpa tanda hubung | `internal/registrasi` |
| Frontend — folder | `nama-modul`, `kebab-case` | `src/modules/registrasi` |

---

## 3. Berkas yang berganti nama

### Backend

| Sebelum | Sesudah |
|---|---|
| `platform/clock/jam.go` · `sistem.go` · `zona.go` | `clock.go` · `system.go` · `zone.go` |
| `auth/identitas.go` · `pengguna.go` · `sesi.go` | `identity.go` · `user.go` · `session.go` |
| `auth/http/galat.go` · `rute.go` · `rute_test.go` | `errors.go` · `routes.go` · `routes_test.go` |
| `auth/provider/berantai.go` · `lokal.go` · `tiruan.go` (+ uji) | `chain.go` · `local.go` · `fake.go` (+ uji) |
| `auth/repo/sqlstore/kueri.go` · `pengguna.{go,sql}` · `sesi.{go,sql}` · `warisan.{go,sql}` | `query.go` · `user.{go,sql}` · `session.{go,sql}` · `legacy.{go,sql}` |
| `auth/usecase/masuk.go` (+ uji) | `login.go` (+ uji) |
| `portal/http/rute.go` | `routes.go` |
| `registrasi/alur.go` · `alur_register.go` · `alur_test.go` | `flow.go` · `flow_register.go` · `flow_test.go` |
| `registrasi/klaim.go` · `tugas.go` · `unitkerja.go` · `validasi.go` (+ uji) | `claim.go` · `task.go` · `unit_of_work.go` · `validation.go` (+ uji) |
| `registrasi/http/galat.go` · `rute.go` | `errors.go` · `routes.go` |
| `registrasi/repo/sqlstore/eksekutor.go` · `klaim.{go,sql}` · `kueri.go` · `pendukung.{go,sql}` · `tugas.{go,sql}` | `executor.go` · `claim.{go,sql}` · `query.go` · `support.{go,sql}` · `task.{go,sql}` |
| `registrasi/repo/memory/pendukung.go` | `support.go` |
| `registrasi/usecase/layanan.go` · `mulai.go` · `tugas.go` · `registrasi_test.go` | `service.go` · `start.go` · `task.go` · `registration_test.go` |
| `cmd/claimpnc/periksa.go` · `registrasi.go` | `check.go` · `registration.go` |
| `migrations/0001_pengguna_dan_sesi.*` | `migrations/0001_user_and_session.*` |
| `migrations/0002_registrasi_klaim_dan_tugas.*` | `migrations/0002_claim_and_task.*` |

> **Mengganti nama berkas migrasi aman.** `golang-migrate` mencatat **nomor versi**, bukan nama
> berkasnya, sehingga basis data yang sudah menjalankan `0001` tetap mengenalinya.

### Frontend

| Sebelum | Sesudah |
|---|---|
| `api/klien.ts` · `api/tipe.ts` | `api/client.ts` · `api/types.ts` |
| `app/Kerangka.tsx` · `PenjagaSesi.tsx` · `PeringatanSesi.tsx` · `sesi.ts` | `Shell.tsx` · `SessionGuard.tsx` · `SessionWarning.tsx` · `session.ts` |
| `components/KolomIsian.tsx` · `PesanGalat.tsx` | `FormField.tsx` · `ErrorMessage.tsx` |
| `modules/masuk/HalamanMasuk.tsx` (+ uji) | `modules/login/LoginPage.tsx` (+ uji) |
| `modules/beranda/HalamanBeranda.tsx` | `modules/home/HomePage.tsx` |
| `modules/portal/PemilihPortal.tsx` (+ uji) | `modules/portal/PortalSelector.tsx` (+ uji) |
| `modules/registrasi/HalamanInbox.tsx` (+ uji) | `modules/registrasi/InboxPage.tsx` (+ uji) |
| `modules/registrasi/HalamanKlaim.tsx` (+ uji) | `modules/registrasi/ClaimPage.tsx` (+ uji) |
| `modules/registrasi/JalurTahap.tsx` · `tipe.ts` | `StagePath.tsx` · `types.ts` |
| `gaya.css` | `styles.css` |

---

## 4. Istilah domain — padanan yang dipakai

Diambil dari `CONTEXT.md`. Dipakai **konsisten di kedua sisi**, sehingga tipe Go dan tipe
TypeScript yang menggambarkan hal yang sama bernama sama.

| Indonesia | Inggris | Indonesia | Inggris |
|---|---|---|---|
| Klaim | `Claim` | Polis | `Policy` |
| Objek Pertanggungan | `InsuredItem` | Coverage | `Coverage` |
| Spreading | `Spreading` | Pelapor | `Reporter` |
| Tugas | `Task` | Tahap | `Stage` |
| Alur | `Flow` | Keputusan | `Decision` |
| Penugasan | `Assigner` | Penerima tugas | `Assignee` |
| Antrean | `Queue` | Workbasket | `Workbasket` |
| Pengguna | `User` | Sesi | `Session` |
| Identitas | `Identity` | Kredensial | `Credential` |
| Profil | `Profile` | Peran | `Roles` |
| Pelanggaran | `Violation` | Galat validasi | `ValidationError` |
| Penyebab Kerugian | `CauseOfLoss` | Tanggal Kejadian | `DateOfLoss` |
| Tanggal Lapor | `ReportDate` | Tanggal Terima Dokumen | `DateReceived` |
| Nilai Estimasi | `EstimateValue` | Kurs | `ExchangeRate` |
| Uang (sen) | `Money` | Persentase ×10⁴ | `Percent` |
| Penerbit Nomor | `NumberIssuer` | Jejak Audit | `AuditTrail` |
| Pemberitahuan | `Notification` | Unit Kerja (transaksi) | `UnitOfWork` |
| Lini Bisnis | `LineOfBusiness` | Jenis Treaty | `TreatyKind` |

Awalan yang dipakai berulang: `Jenis…` → `…Kind` · `Kode…` → `Code…` · `Respons…` →
`…Response` · `Permintaan…` → `…Request` · `Isian…` → `…Input` / `…FormValues` ·
`Ambil…` → `Get…` · `Simpan…` → `Save…` · `gunakan…` (hook React) → `use…`.

---

## 5. Kasus yang menuntut perhatian

Enam hal berikut **tidak** dapat diselesaikan dengan penggantian nama biasa, dan dua di antaranya
sempat menyebabkan kerusakan yang tertangkap perkakas sebelum masuk ke repo.

| # | Kasus | Perlakuan |
|---|---|---|
| 1 | **Kata Indonesia berawal kapital berbentuk sama dengan nama tipe** — `Kode`, `Sesi`, `Klaim`, `Polis` | Di komentar, hanya nama **berpunuk** (`KlaimRepo`, `TanggalKejadian`) yang diganti. Nama satu kata diganti hanya bila ia kata pertama komentar dokumentasi yang menamai deklarasi di bawahnya |
| 2 | **Teks JSX adalah teks yang dilihat pengguna** — `<button>Simpan</button>` | Wilayah teks JSX dipisahkan dari wilayah kode dan tidak pernah disentuh. Seluruh 68 potongan teks dibandingkan sebelum dan sesudah, dan identik |
| 3 | **Literal regex di berkas uji memuat teks UI** — `findByText(/Tidak ada pekerjaan/)` | Diperlakukan seperti string |
| 4 | **Interpolasi template adalah kode** — `` `${nama}` `` | Diperlakukan sebagai kode; menandainya string akan membuat deklarasi berganti sementara pemakaiannya tertinggal |
| 5 | **Nama properti DTO frontend adalah kontrak JSON** — `nama`, `klaim`, `tugas`, `kode` | Dikeluarkan dari peta. Variabel lokal yang kebetulan bernama sama ikut tetap, lalu diganti satu per satu |
| 6 | **Nama parameter rute hidup di dalam string** — `path="/registrasi/klaim/:klaimID"` | Ia terikat pada destrukturisasi `useParams`, jadi ia nama di dalam kode. Diganti menjadi `:claimID`. Segmen jalurnya sendiri (`/registrasi/klaim`) tetap Indonesia |

Dua rujukan pengenal lain juga hidup di dalam string dan harus ikut berganti:
`Pick<SessionState, 'expiresAt'>` di `app/session.ts`, dan `register('username')` beserta
`id="username"` di `modules/login/LoginPage.tsx`.

---

## 6. Yang dibuktikan, bukan diasumsikan

Empat kontrak diperiksa dengan membandingkan sidik ringkas isi sebelum dan sesudah:

| Kontrak | Hasil |
|---|---|
| Seluruh tag `json:"…"` di backend | **identik** |
| Seluruh jalur URL `/api/…` | **identik** |
| Seluruh isi berkas `.sql` dan migrasi | **identik** |
| Seluruh nama variabel lingkungan | **identik** |
| Seluruh teks JSX yang dilihat pengguna | **identik** — 68 potongan, sebelum dan sesudah |
