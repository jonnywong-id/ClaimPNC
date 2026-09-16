# Deployment Strategy & Infrastructure

Mengacu pada **VM/bare metal on-premise** (D-08) dengan tuntutan **ketersediaan 24/7** (D-27),
untuk 200–300 pengguna aktif harian.

---

## 1. Topologi

```
                    ┌────────────────────┐
                    │   Load Balancer    │  health check aktif
                    │  (nginx / HAProxy) │  sticky session TIDAK diperlukan
                    └─────────┬──────────┘
              ┌───────────────┴───────────────┐
    ┌─────────▼─────────┐          ┌──────────▼────────┐
    │  VM Aplikasi 1    │          │  VM Aplikasi 2    │
    │                   │          │                   │
    │  binary Go        │          │  binary Go        │
    │  + SPA tersemat   │          │  + SPA tersemat   │
    │  (satu proses)    │          │  (satu proses)    │
    └─────────┬─────────┘          └──────────┬────────┘
              └───────────────┬───────────────┘
                    ┌─────────▼──────────┐
                    │  Oracle 19c        │
                    │  → PostgreSQL 17+  │
                    │  (dibagi dgn Pega  │
                    │   selama paralel)  │
                    └────────────────────┘
```

**Minimal dua instance aplikasi** — bukan pilihan, melainkan konsekuensi langsung dari D-27.
Dengan satu instance, setiap deployment berarti downtime.

**Satu proses per VM.** Binary Go menyajikan API sekaligus berkas statis SPA. Tidak ada
Node.js, tidak ada web server terpisah untuk berkas statis. Ini konsekuensi dari menolak
Next.js/Nuxt di D-23, dan inilah manfaat nyatanya: satu hal untuk di-deploy, dipantau, dan
ditambal keamanannya.

---

## 2. Syarat teknis untuk 24/7

| Syarat | Alasan |
|---|---|
| **Aplikasi stateless** | Session di memori satu instance akan hilang saat instance itu di-restart, dan pengguna terlempar keluar di tengah pekerjaan |
| **Graceful shutdown** | Saat instance dimatikan, permintaan yang sedang berjalan harus diselesaikan lebih dulu, bukan diputus |
| **Health check `live` dan `ready` terpisah** | Load balancer harus tahu kapan instance baru benar-benar siap menerima trafik |
| **Migrasi skema backward-compatible** | Saat rolling deployment, versi lama dan baru berjalan bersamaan terhadap skema yang sama (DB-7) |
| **Migrasi dijalankan terpisah dari start aplikasi** | Bila dijalankan saat start, dua instance akan bermigrasi bersamaan |
| **Job terjadwal hanya berjalan di satu instance** | Tanpa ini, job berjalan dua kali. Ditegakkan dengan kunci di database, bukan konfigurasi manual |

Poin terakhir sering terlewat dan akibatnya serius: job pengiriman notifikasi atau transfer ke
kasir yang berjalan dua kali berarti pengiriman ganda.

---

## 3. Rolling deployment

1. Migrasi skema dijalankan (backward-compatible, aman terhadap versi lama).
2. Instance 1 ditandai *tidak siap*; load balancer berhenti mengirim trafik.
3. Instance 1 menyelesaikan permintaan yang sedang berjalan, lalu berhenti.
4. Instance 1 dijalankan dengan versi baru; menunggu sampai `/health/ready` sukses.
5. Load balancer mengembalikan trafik ke Instance 1.
6. Ulangi untuk Instance 2.

**Tanpa downtime.** Selama proses, satu instance selalu melayani.

**Bila gagal:** kembalikan binary ke versi sebelumnya dan ulangi langkah yang sama. Karena
migrasi skema backward-compatible, versi lama tetap berfungsi terhadap skema baru.

---

## 4. Lingkungan

| Lingkungan | Kegunaan | Database |
|---|---|---|
| **Development** | Pengembangan lokal | Database lokal |
| **Staging** | Pengujian, **verifikasi kesetaraan dengan Pega** | **Salinan produksi apa adanya, tanpa penyamaran** (`D-64`) |
| **Production** | Operasional | Oracle produksi (dibagi dengan Pega selama paralel) |

**Staging wajib memakai salinan data produksi.** Verifikasi kesetaraan hanya bermakna bila
dijalankan terhadap data nyata — data uji buatan tidak akan memicu kasus tepi yang justru paling
sering menjadi sumber perbedaan hasil. Buktinya konkret: cacat yang ditemukan pada verifikasi
Fase 1 — toleransi spreading berupa pencocokan substring, kurs yang mengembalikan `1`,
`IDSALVAGE = NULL` — **muncul dari data nyata yang tidak akan terpikir dibuat**.

> **Koreksi v2.0 — data tidak disamarkan.** Versi sebelumnya mewajibkan penyamaran NIK, nomor
> telepon, alamat, data medis, dan nomor rekening. **`D-64` memutuskan sebaliknya:** data disalin
> **apa adanya**, dan **hak akses lingkungan staging diperketat** sebagai kontrol penggantinya.
> Penyamaran akan menghilangkan justru sifat data yang membuatnya berguna untuk uji kesetaraan.

**Konsekuensi yang diterima secara sadar:**

1. **Staging memuat data nasabah nyata** — nomor polis, nama tertanggung, NPWP, nomor rekening,
   dan **data medis** pada lini PA dan Travel.
2. **Klasifikasi staging naik setara produksi** untuk keperluan keamanan.
3. Pembatasan akses data medis (`FR-R2`) berlaku juga di staging, bukan hanya produksi.

**Masih terbuka:** siapa yang menyetujui akses staging, berapa lama salinan disimpan, dan prosedur
pemusnahannya — ditujukan ke pihak yang sama dengan `D-40`.

**Satu prasyarat yang belum dikonfirmasi:** uji kesetaraan menuntut **Pega staging yang dapat
ditembak dari luar** sebagai pembanding. Ketersediaannya **belum dipastikan**, dan tanpa itu
gerbang 1 seluruh modul tidak dapat dijalankan (`ADR-0027`).

**Satu risiko yang berlaku pada penyalinannya sendiri:** `R-12` — pergeseran zona waktu — berlaku
pada **proses menyalin data ke staging**, bukan hanya pada migrasi akhir. Salinan yang bergeser
akan menghasilkan selisih palsu di setiap pengujian.

---

## 5. Rilis

- **Artefak:** satu berkas binary Go dengan SPA tersemat, ditandai versi dan commit.
- Binary yang sama dipromosikan dari staging ke production — **tidak pernah dibangun ulang**
  untuk production. Build ulang berarti yang diuji bukan yang dijalankan.
- Konfigurasi berasal dari lingkungan, bukan dari binary.
- Setiap rilis dicatat: versi, commit, isi perubahan, waktu, pelaksana.

---

## 6. Backup dan pemulihan

Mengikuti **standar backup korporat data center Sinarmas** (D-29).

**Yang harus diperiksa terhadap standar itu**, dan diangkat ke tim infra bila belum sejalan:

| Pertanyaan | Kenapa penting |
|---|---|
| Apakah RPO standar sejalan dengan sifat data klaim? | Data klaim menyangkut nilai uang dan kewajiban ke tertanggung |
| Apakah RTO standar sejalan dengan tuntutan 24/7 (D-27)? | Standar backup menjawab pemulihan bencana, bukan operasional harian — keduanya berbeda |
| Apakah pemulihan pernah benar-benar diuji? | Backup yang tidak pernah diuji pemulihannya bukan backup |
| Apakah retensi memenuhi kewajiban audit (D-28)? | Retensi audit bisa lebih panjang dari retensi backup biasa |

---

## 7. Infrastruktur yang diperlukan

| Komponen | Keterangan |
|---|---|
| 2 VM aplikasi | CPU dan memori sedang; Go hemat sumber daya |
| 1 load balancer | nginx atau HAProxy dengan health check aktif |
| Database | Oracle 19c yang ada; kelak PostgreSQL 17+ (D-24) |
| Sertifikat TLS | Wajib; termasuk untuk sambungan internal |
| Pengumpul log | Terpusat agar log dari kedua instance dapat dicari bersamaan |
| Pemantauan | Prometheus + Grafana bila tersedia; minimal pemantauan health check |
| Penyimpanan dokumen | API internal yang sudah ada (D-16) — tidak perlu infrastruktur baru |

**Yang sengaja tidak diperlukan:** Kubernetes (D-08), message broker, cache terdistribusi,
service mesh. Seluruhnya dijelaskan di Future Architecture §6.

---

## 8. Hidup berdampingan dengan Pega

Selama masa paralel (D-05):

- Pega dan aplikasi Go berjalan bersamaan, memakai database yang sama (D-21).
- Pengguna diarahkan ke sistem yang tepat berdasarkan modul — lewat menu portal atau aturan
  routing di load balancer.
- **Pega tidak boleh dinonaktifkan sebelum Tahap 7** (Migration Strategy §2). Kemampuan mundur
  bergantung sepenuhnya pada Pega yang masih berfungsi penuh.
- Beban database naik karena dua sistem membacanya. Ini harus dipantau sejak awal masa paralel,
  bukan setelah pengguna mengeluh.
- **Perubahan skema menempuh tiga pihak** — permintaan tertulis tim pengembang, persetujuan Work
  Owner, pelaksanaan DBA, lalu **wajib diuji dengan menjalankan Pega dan Go bersamaan** (`D-63`).
  Satu `ALTER` yang keliru pada tabel yang dibaca 116 rule Pega menghentikan produksi.

---

## 9. Job terjadwal pada dua instans — belum diputuskan

Sistem lama memiliki **5 job terjadwal + 1 agent**, seluruhnya `pyApplicableTo=Cluster` —
mekanisme Pega yang menjamin satu job berjalan **sekali saja** meski ada banyak node (`D-57`).

Go **tidak punya padanan otomatis**, sementara §1 menuntut minimal dua instans di belakang load
balancer. Tiga pilihan dipertimbangkan di `ADR-0022` dan **belum diputuskan**: penjadwal di dalam
binary dengan penguncian lewat database, cron sistem operasi di satu VM, atau instans penjadwal
terpisah dari binary yang sama.

> **Kenapa ini bukan detail teknis kecil.** Bila penguncian salah dirancang,
> **`AutoAcceptKomite` dapat berjalan dua kali** — menyetujui komite dua kali pada klaim yang
> sama. Itu kesalahan bernilai uang, bukan gangguan operasional.

Dua pertanyaan lain menunggu Work Owner, bukan tim teknis: apakah `AutoAcceptKomite` yang
menyetujui komite otomatis tiap hari **jam 06:00 tanpa pengguna sama sekali** dipertahankan, dan
apakah `pyBypassActivityAuthentication=true` pada agent diterima.

**Bagaimana kegagalan job diketahui** — notifikasi, catatan, atau dashboard — juga belum
ditetapkan; di sistem lama tidak ada bukti mekanisme apa pun untuk itu.
