---
title: "TKT-F4-003 — Master Penerima Notifikasi (mailbox fungsional)"
labels: [modul::F-4, tipe::migrasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F4-003 — Master Penerima Notifikasi (mailbox fungsional)

Status: ready-for-human
Kesiapan: siap
Modul: F-4 · Gelombang: 1 · Bergantung pada: TKT-F4-001
Requirement: FR-F4    Keputusan: D-15, D-67    ADR: 0025    Risiko: R-17
Rule Pega yang digantikan: **66 alamat email unik** yang tertanam di activity — di antaranya `Activity/InputRegister_act-Act.xml:16154` (7 email pimpinan dalam satu string), `:16297` (UW FIRE), `:16461` (UW ANEKA), `:16628` (UW MARINE); akun Gmail pribadi di `Activity/AutoEmailDownloadProposeAdjustment-Act.xml:7872`, `:8031`, `Activity/SetStsSalvagePNC_act-Act.xml:694`, `Activity/LetterOfAssignment2_Act-Act.xml:7725`, `Activity/SendEmailUnprotectPremi-Act.xml:4823`
Peran penguji gerbang 2: **User Admin** dan **PncManagerAdmin** (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Penerima notifikasi menjadi master data berupa **mailbox fungsional**, dan **tidak ada satu pun
akun pribadi** yang dibawa ke sistem baru.

Nilai bisnisnya langsung: hari ini, notifikasi klaim dikirim ke **≥6 alamat Gmail pribadi di jalur
produksi**. Ketika orangnya pindah atau keluar, notifikasi itu **tetap terkirim ke sana** dan tidak
ada yang menyadarinya sampai sesuatu tidak ditindaklanjuti.

## Ruang lingkup

- Master penerima: per **jenis peristiwa** dan per **Group Panel**, berisi alamat tujuan dan
  tembusan.
- Pemetaan 66 alamat lama ke mailbox fungsional pengganti — dikerjakan bersama Work Owner saat
  master diisi.
- **Pembersihan kategori khusus**: lima alamat Gmail yang dipakai sebagai **Operator ID** di filter
  laporan KPI — itu masalah **identitas**, bukan notifikasi, dan menyentuh `F-3` sekaligus `F-4`.
- Penghapusan blok `// TESTING` yang menimpa email produksi.

## Non-goal

- **Tidak** membangun pengirim notifikasi — itu `S-3`.
- **Tidak** memutuskan isi pesan notifikasi.
- **Tidak** memindahkan kredensial SMTP — itu `D-40`, masih `OPEN`.

## Acceptance criteria

- [ ] Master memuat penerima per jenis peristiwa dan per Group Panel, dan **seluruh 66 alamat
      lama terpetakan** — dihitung dan dilaporkan angkanya; alamat yang sengaja tidak dibawa
      dicatat alasannya.
- [ ] **Nol alamat email tertanam di kode** — diuji pemindaian pola; build gagal bila muncul.
- [ ] **Nol akun pribadi** di master — diuji terhadap daftar domain yang diizinkan; alamat di luar
      domain korporat ditolak saat disimpan.
- [ ] Lima alamat yang dipakai sebagai Operator ID **tidak** masuk master penerima; ia dilaporkan
      sebagai temuan untuk `F-3` — diuji bahwa keduanya tidak tertukar.
- [ ] Mengubah penerima **tidak menuntut deployment** — diuji dengan mengubah master dan memeriksa
      pengiriman berikutnya memakai alamat baru.
- [ ] Perubahan master tercatat di jejak audit (`TKT-F4-001`).

## Dependency / Blocked by

Bergantung pada `TKT-F4-001`. Pengisian alamat pengganti dikerjakan bersama Work Owner — **itu
bagian pelaksanaan, bukan penghalang**.

## Constraint keamanan, data, operasional

- **Alamat email tidak pernah ditulis lengkap di dokumen yang di-commit** (`D-69`) — di tiket ini
  pun alamatnya dirujuk dengan `berkas:baris`, bukan nilainya.
- Blok `// TESTING` pada `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141`
  **menimpa email produksi dengan precondition yang identik** — ia tidak boleh punya padanan apa
  pun di sistem baru.
- Kredensial SMTP **tidak** disimpan di master ini; tempatnya menunggu `D-40` (`R-17`).

## Migrasi skema / rollout / rollback

Menambah tabel master — tidak menyentuh tabel Pega. Selama masa paralel, **Pega tetap memakai
alamat hardcode-nya sendiri**; keduanya hidup berdampingan sampai `S-3` pindah.

**Rollback:** aplikasi Go kembali ke daftar bawaan; **tidak kembali ke akun pribadi** — daftar
bawaan yang dipakai adalah mailbox fungsional.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
grep -rInE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}" internal/ --include=*.go   # HARUS 0
go test ./internal/app/masterdata/... -run TestPenerimaNotifikasi
go test ./internal/app/masterdata/... -run TestTolakDomainDiLuarKorporat
go run ./cmd/tools/banding-penerima daftar-lama.csv   # laporkan yang belum terpetakan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 66 alamat email unik | `D-15` terverifikasi · `docs/verifikasi-bukti-adr.md` §7.2 |
| ≥6 akun Gmail pribadi di jalur produksi | `D-67` · lokasi tercantum di header tiket |
| 5 alamat dipakai sebagai Operator ID di filter KPI | `D-67` — menyentuh `F-3` sekaligus `F-4` |
| Blok `// TESTING` menimpa email produksi | `Activity/InputRegister_act-Act.xml:16693`, `:16830`, `:16998`, `:17141` |
| Seluruh penerima dari master, mailbox fungsional | `D-67` |

## Comments
