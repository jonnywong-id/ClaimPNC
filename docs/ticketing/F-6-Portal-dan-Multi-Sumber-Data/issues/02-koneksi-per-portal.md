---
title: "TKT-F6-002 — Koneksi dan pool per portal"
labels: [modul::F-6, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F6-002 — Koneksi dan pool per portal

Status: needs-info
Kesiapan: **terhalang `D-78`** — di mana kewenangan portal disimpan.
Penomoran klaim tertutup (`D-76`); jumlah portal tertutup (Work Owner 2026-09-19: **pool
mengikuti database yang ada**, bukan angka tetap — sudah terpasang, lihat "Keadaan kode").
Modul: **F-6 Portal & Multi-Sumber Data** · Gelombang: 1 · Bergantung pada: TKT-F6-001, TKT-F2-001
Requirement: FR-F6    Keputusan: D-75, D-71, D-76    ADR: 0030, 0004    Risiko: R-20
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama memakai satu koneksi dan membedakan perilaku lewat hostname
Peran penguji gerbang 2: **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Setiap portal punya **pool koneksinya sendiri**, dan seluruh pembacaan serta penulisan mengikuti
portal yang sedang aktif.

Nilai bisnisnya adalah alasan `D-75` menolak database bersama: dengan koneksi terpisah, **pemisahan
data menjadi sifat struktural**. Portal A tidak dapat membaca data portal B karena koneksinya memang
berbeda — bukan karena setiap kueri ingat menyaring. Kelas kesalahan "ada satu kueri yang lupa
menyaring" **hilang seluruhnya**, dan kelas itu tidak dapat diuji habis.

## Ruang lingkup

- Pool koneksi per portal, dibuat dari konfigurasi `TKT-F6-001`.
- Seam Repository (`TKT-F2-001`) mengambil koneksi **dari portal aktif**, bukan dari variabel global.
- Batas jumlah koneksi **per portal**, bukan satu batas dibagi rata.
- Pool laporan terpisah (`TKT-F2-001`) tetap berlaku — **per portal**.
- Perilaku bila database satu portal mati: portal lain **tetap berjalan**.

## Non-goal

- **Tidak** menyatukan data antar portal dengan cara apa pun.
- **Tidak** membangun laporan lintas portal — `D-75` menetapkan laporan **per portal**.

## Yang sudah ditegaskan Work Owner (2026-09-19)

| Pernyataan | Akibatnya bagi tiket ini |
|---|---|
| **Setiap portal terkoneksi ke database masing-masing** | Rancangan `D-75` dikonfirmasi; tidak ada lagi pertanyaan "apakah satu database bersama cukup" |
| **Pool dijalankan sesuai portal** | Ruang lingkup butir 1–3 dikonfirmasi apa adanya |
| **Tiap server punya `POOLDATA` sendiri**, dan aplikasi membaca milik server tempat ia berjalan | **Isi master pun berbeda antar portal** — lihat constraint di bawah. Ini yang menaikkan tiket dari "kerapian" menjadi "kebenaran data" |
| **Pool mengikuti database yang ada** — bukan angka portal yang ditetapkan di muka | Pertanyaan "berapa portal" **tertutup**: jawabannya bukan angka melainkan aturan. Perilakunya **sudah terpasang** — lihat "Keadaan kode" |

## Keadaan kode hari ini — yang sudah ada dan yang belum

Diperiksa 2026-09-19. Pengerjaan tiket ini **tidak dimulai dari nol**:

| Bagian | Status | Letak |
|---|---|---|
| Pool per portal, dibuka dari konfigurasi | **sudah ada** | `internal/platform/db` — `KumpulanBaru`, `Untuk(alias)`, `Tersedia()`, `Utama()` |
| **Pool mengikuti database yang ada** | **sudah ada** | daftar alias dipindai dari variabel `POOLDATA_<ALIAS>_HOST` (tidak ada daftar portal di kode, `D-15`); `parameterPortal` melewati yang konfigurasinya tidak lengkap; `KumpulanBaru` melewati yang gagal dibuka dan mencatatnya — **kecuali portal utama, yang gagalnya fatal** |
| Portal yang databasenya mati **terlihat pengguna**, bukan hanya di log | **sudah ada** | `Tersedia()` → `aliasSiap` → `GET /api/portal` → `Portal.siap`; layar menandainya "belum tersedia", bukan menyembunyikannya |
| Portal yang sedang dipilih, bertahan melewati muat ulang | **sudah ada** | `frontend/src/app/portal.ts` — `gunakanPortalTerpilih`, sessionStorage |
| Daftar portal beserta kesiapannya | **sudah ada** | `internal/portal` + `GET /api/portal` |
| **Portal ikut pada setiap permintaan** | **BELUM ADA** | `frontend/src/api/klien.ts` tidak pernah mengirim portal |
| **Server memilih pool dari portal aktif** | **BELUM ADA** | seluruh repo memakai `Utama()` |

Jadi yang hilang tepat **satu mata rantai**: portal tidak pernah menyeberang dari peramban ke
server.

> **Peringatan bagi yang mengerjakan.** Mata rantai itu tampak sepele — kirim satu header, panggil
> `Untuk(alias)` — dan justru **di situ letak `R-20`**. Memilih pool dari alias yang dikirim klien
> **tanpa memeriksa kewenangan pengguna atas portal itu** membuat siapa pun dapat membaca data badan
> hukum lain hanya dengan mengganti satu header. Itu lebih buruk daripada keadaan sekarang.
>
> Pemeriksaan kewenangannya sendiri **belum dapat ditulis**, karena `D-78` meninggalkan pertanyaan
> di mana kewenangan portal disimpan: login sama untuk keempat entitas, sehingga data "siapa berhak
> atas portal mana" bersifat lintas portal dan tidak dapat tinggal di dalam database tiap portal —
> memeriksa hak atas portal B menuntut membaca database B sebelum penggunanya terbukti berhak.
>
> Urutan yang benar karena itu: `D-78` dijawab → `TKT-F6-003` (kewenangan dinilai ulang di server)
> → baru pemilihan pool di tiket ini.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Kenapa menahan |
|---|---|---|
| **Di mana kewenangan portal disimpan** (`D-78`) | **Work Owner + Keamanan Informasi** | Tanpa itu, memilih pool dari alias kiriman klien adalah kebocoran `R-20`, bukan perbaikan. **Inilah satu-satunya penghalang yang tersisa** |
| **Batas koneksi per portal berapa** | **Work Owner + Tim Infra** | Mekanismenya sudah ada (`POOLDATA_<ALIAS>_MAKS_KONEKSI`, bawaan 20/5/30m); yang belum ada angkanya. Tidak menahan pengerjaan — menahan penyetelan sebelum produksi |

**Sudah tertutup:**

- ~~**Berapa portal**, dan apakah keempat database sudah tersedia~~ — Work Owner, 2026-09-19:
  jawabannya bukan angka melainkan aturan, yaitu **pool mengikuti database yang ada**. Sudah
  terpasang; database yang belum siap dilewati, dicatat, dan ditandai "belum tersedia" di layar.
- ~~**Nomor klaim unik lintas portal?**~~ — `D-76`: tidak perlu penanda portal; nomor unik **di
  dalam portal**, tidak dijamin unik antar portal. Konsekuensinya: menyebut nomor klaim tanpa
  portalnya menjadi **ambigu**, termasuk pada LOD/PLA/DLA yang keluar ke pihak luar.

> **Penomoran klaim sudah diputuskan** (`D-76`): tanpa penanda portal. Bila kelak diputuskan ada
> laporan konsolidasi lintas portal, keputusan itu **harus ditinjau ulang lebih dulu** — menambah
> penanda setelah klaim terbit menuntut migrasi data.

## Acceptance criteria

- [ ] Setiap portal punya pool sendiri, dan jumlah pool **sama dengan jumlah portal aktif** — diuji.
- [ ] Kueri yang dijalankan saat portal A aktif **tidak pernah menyentuh database portal B** —
      diuji dengan pencatatan koneksi pada 20 operasi berbeda: **nol kebocoran**.
- [ ] **Database satu portal mati tidak menjatuhkan portal lain** — diuji dengan mematikan satu
      database: portal lain tetap melayani.
- [ ] Batas koneksi ditegakkan **per portal**; satu portal yang sibuk **tidak menghabiskan jatah**
      portal lain — diuji dengan beban pada satu portal.
- [ ] Laporan berat di satu portal tidak menghabiskan koneksi transaksi portal itu maupun portal
      lain (`TKT-F2-001`) — diuji.
- [ ] Tidak ada jalur kode yang dapat memilih koneksi **tanpa portal aktif** — diuji: permintaan
      tanpa portal ditolak, bukan jatuh ke default.
- [ ] **Master data dibaca dari database portal aktif, bukan portal utama** — diuji pada kasus yang
      isinya benar-benar berbeda: membuka Ambang Komite di portal Simasnet mengembalikan baris
      ber-`TYPE_BUSINESS` `SIMASNET`, dan di portal ASM mengembalikan NONMBU/PA/TRAVEL/BONDING.
      Keduanya dibandingkan dalam satu sesi tanpa login ulang.
- [ ] Gerbang 2: UAT **PncAdmin** pada minimal dua portal.

## Dependency / Blocked by

`TKT-F6-001` · `TKT-F2-001` · `TKT-F2-006` (nomor klaim). **Terhalang Work Owner dan Tim Infra.**

## Constraint keamanan, data, operasional

- **Jatuh ke koneksi default saat portal tidak diketahui adalah kebocoran data antar badan hukum.**
  Perilaku yang benar adalah menolak permintaan, bukan menebak portalnya.
- Empat pool melipatkan kebutuhan koneksi di sisi database. Bila kapasitasnya tidak disiapkan,
  portal keempat akan gagal saat portal lain sedang sibuk.
- Kredensial keempat database tunduk `D-40` dan `R-17` — dari penyimpanan rahasia, tidak pernah
  masuk repositori.
- `P-1` (satu tabel ditulis satu sistem) kini berlaku **di dalam tiap database**, dan tetap berlaku
  penuh selama Pega dan Go berjalan berdampingan (`D-05`) — **di keempat entitas**.
- **Master data pun berbeda isinya antar portal, dan itu sudah terbukti — bukan dugaan.** Work Owner
  menegaskan 2026-09-19 bahwa tiap server punya `POOLDATA`-nya sendiri dan aplikasi membaca milik
  server tempat ia berjalan. Contoh nyatanya: `POOLDATA.EMAILKOMITE` pada server ASM memuat
  `TYPE_BUSINESS` NONMBU, PA, TRAVEL, dan BONDING, sementara baris `SIMASNET`/`SIMASNETA` hanya ada
  di server Simasnet. Aturan penjenjangan komitenya pun berbeda — Simasnet memilih **satu** penyetuju
  secara acak, yang lain **kumulatif**.
- **Membaca master dari koneksi utama menghasilkan angka yang salah tanpa satu pun tanda.** Modul
  `B-7` hari ini memasang repo ambang komite pada koneksi utama
  (`cmd/claimpnc/main.go`, dicatat di sana dan di `komite/repo/sqlstore/ambang.sql`). Begitu portal
  kedua dilayani, layar akan menampilkan jenjang persetujuan **milik entitas lain**: seluruh namanya
  masuk akal, tidak ada galat, dan yang keliru hanya *siapa* yang berwenang menyetujui uang. Bentuk
  kegagalannya sama dengan `R-20`, dan tiket inilah yang menutupnya.

## Migrasi skema / rollout / rollback

Tidak mengubah tabel klaim. Menambah kebutuhan konfigurasi per portal.

**Rollout:** satu portal pada satu waktu. Portal yang belum dialihkan tetap memakai Pega.

**Rollback:** mengembalikan lalu lintas portal itu ke Pega. Rollback **per portal**, bukan seluruhnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/db/... -run TestPoolPerPortal
go test ./internal/adapter/db/... -run TestKueriTidakBocorAntarPortal
go test ./internal/adapter/db/... -run TestDatabaseSatuPortalMati
go test ./internal/adapter/db/... -run TestTanpaPortalDitolakBukanDefault
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Satu database per entitas | `D-75` · `ADR-0030` |
| Alasan menolak database bersama berkolom entitas | `ADR-0030` bagian Opsi |
| Pool laporan terpisah | `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` butir 7 |
| Nomor klaim `PNCN.YY.xxxx` dari sequence, tanpa penanda portal | `D-71` · `D-76` |
| Satu tabel ditulis satu sistem | `P-1` · `docs/Steering/07-MIGRATION-STRATEGY.md:15` |

## Comments
