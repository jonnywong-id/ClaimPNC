---
title: "TKT-B06-002 — Aturan routing dan pembagian beban"
labels: [modul::B-6, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::3]
milestone: "Gelombang 3 — Jalur klaim inti"
epic: "Migrasi Claim PNC"
---

# TKT-B06-002 — Aturan routing dan pembagian beban

Status: needs-info
Kesiapan: **terhalang artefak** — tiga router tidak ada di export
Modul: **B-6 Penugasan & Inbox** · Gelombang: 3 · Bergantung pada: TKT-B06-001
Requirement: FR-B6    Keputusan: D-26, D-15    ADR: 0019, 0025    Risiko: R-04, R-16
Rule Pega yang digantikan: 8 router — `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`, `ToWorkList`, `ToWorkbasket` · algoritma beban pada `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` dan `AddTJobCounterPIC_SQL`
Peran penguji gerbang 2: **PncManagerAdmin** — peran yang paling merasakan bila pembagian beban timpang

## Hasil yang diharapkan (dan nilai bisnisnya)

Tugas baru diberikan kepada petugas yang **paling sedikit bebannya**, bukan acak dan bukan selalu
orang yang sama.

Nilai bisnisnya langsung terasa: pembagian beban yang timpang berarti sebagian petugas menumpuk
pekerjaan sementara yang lain menganggur — dan itu memanjangkan TAT tanpa alasan yang terlihat di
laporan mana pun.

## Ruang lingkup

- Aturan routing sebagai **data**, bukan kode tersebar: untuk tiap tahap, siapa yang berhak
  menerima dan bagaimana dipilih.
- Algoritma pembagian beban: **`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya
  menang; pencacah dinaikkan setelah penugasan.
- Delapan router lama dibangun ulang sebagai aturan routing terpadu.
- **Penghapusan hardcode nama orang** di dalam SQL routing.

## Non-goal

- **Tidak** menangani penugasan komite (`B-7`).
- **Tidak** menangani penguncian (`TKT-B06-003`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`** — tidak ada di export | **Tim Pega** (`R-04`, `R-16`) | Algoritma bebannya sudah terbaca, tetapi **syarat kelayakan** (siapa yang boleh masuk daftar calon) hanya ada di dalam router |
| **Konfirmasi ketiga router memang memakai jalur `counter_quota`** | **Work Owner + Tim Pega** | Rekonstruksi dari kueri lain **belum diverifikasi** terhadap router aslinya |
| **`operator_ID != 'ELLENSUPRIYATI'` di-hardcode di dalam SQL** — dibawa sebagai peran, atau dihapus? | **Work Owner** | Bila dibawa, ia menjadi baris master; bila dihapus, perilaku penugasan **berubah** dan itu harus dinyatakan sebagai perubahan sadar |
| **3 `Data-Admin-WorkBasket`** — definisi antrean bersama | **Tim Pega** | Menentukan antrean mana yang ada dan siapa yang boleh mengambilnya |

## Acceptance criteria

- [ ] Tugas baru diberikan kepada petugas dengan **`counter_quota` terkecil** — diuji dengan tiga
      petugas berbeda beban: yang terkecil menang.
- [ ] Pencacah beban **naik tepat satu** setelah penugasan, dan **turun** saat tugas selesai —
      diuji; bila hanya naik, pembagian beban menjadi timpang permanen.
- [ ] Dua penugasan **bersamaan** tidak memberikan tugas ke petugas yang sama secara keliru —
      diuji dengan dua proses paralel.
- [ ] **Nol nama orang di kode maupun kueri routing** — diuji pemindaian terhadap 24 Operator ID
      yang diketahui (`ADR-0025`).
- [ ] Aturan routing dapat diubah lewat master **tanpa deployment** — diuji.
- [ ] Petugas yang sedang tidak aktif **tidak menerima tugas baru** — diuji.
- [ ] Gerbang 1: petugas yang terpilih **sama dengan Pega** pada 30 penugasan contoh di staging.
- [ ] Gerbang 2: UAT **PncManagerAdmin** — memeriksa sebaran beban setelah 50 penugasan.

## Dependency / Blocked by

`TKT-B06-001` · `TKT-F4-001` (master aturan routing). **Terhalang Tim Pega dan Work Owner.**

## Constraint keamanan, data, operasional

- Pencacah beban adalah **state yang harus dijaga konsisten**. Bila naik tanpa pernah turun, atau
  gagal naik karena galat, pembagian beban menjadi timpang **secara permanen** dan tidak ada yang
  menyadarinya sampai ada yang mengeluh.
- Hardcode nama orang di dalam SQL adalah bentuk terburuk dari `D-15`: ia **tidak terlihat** saat
  membaca kode aplikasi, hanya saat membaca teks kueri.
- Penugasan tercatat di jejak audit — ia menentukan siapa yang bertanggung jawab atas sebuah klaim.

## Migrasi skema / rollout / rollback

Menambah master aturan routing dan kolom pencacah beban.

**Rollback:** kembali ke router Pega **tidak tersedia** untuk klaim yang sudah dipegang sistem
baru. Rollback berarti menghentikan penugasan baru di sistem baru.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/domain/penugasan/... -run TestPembagianBebanTerkecil
go test ./internal/domain/penugasan/... -run TestPencacahNaikTurun
go test ./internal/app/penugasan/... -run TestPenugasanBersamaan
go run ./cmd/tools/cek-hardcode-operator internal/domain/penugasan/   # HARUS 0
go run ./cmd/s8 banding --modul B-6 --aturan routing --kasus 30
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `ORDER BY counter_quota ASC` | `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` |
| Pencacah dinaikkan `AddTJobCounterPIC_SQL` | `ADR-0019` |
| Delapan router yang dibangun ulang | `D-26` · `ADR-0019` |
| Tiga router tidak ada di export | `R-04` |
| `operator_ID != 'ELLENSUPRIYATI'` hardcode di SQL | `docs/verifikasi-bukti-adr.md` §15 baris `B-6` |

## Comments
