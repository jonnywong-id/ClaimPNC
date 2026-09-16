---
title: "TKT-S4-001 — Klien REST keluar dan penanganan kegagalannya"
labels: [modul::S-4, tipe::migrasi, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-S4-001 — Klien REST keluar dan penanganan kegagalannya

Status: needs-info
Kesiapan: **terhalang keputusan keamanan**
Modul: **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-F1-004
Requirement: FR-S4    Keputusan: D-25    ADR: 0017    Risiko: R-18
Rule Pega yang digantikan: **21 Connect REST** di `Connect REST/` — antara lain `GetPremiumPaid_SPK`, `SendDataPaidASMtoCashier_2`, `SendKlaimToASO`, `ServiceCloseClaimNonMBU`, `ServiceSetDLA`, `UploadDokumenPNC`, `InjectDataRekeningToKasir`
Peran penguji gerbang 2: **PncAdmin** dan pemilik masing-masing integrasi

## Hasil yang diharapkan (dan nilai bisnisnya)

Panggilan keluar ke sistem lain berjalan dari Go, dan **kegagalan sistem lain tidak menjatuhkan
Claim PNC**.

Nilai bisnisnya ada pada kalimat kedua. Sistem yang dipanggil dimiliki pihak lain dan akan mati
tanpa memberi tahu. Bila petugas tidak dapat menyimpan registrasi hanya karena layanan premi sedang
tidak menjawab, kegagalan pihak lain menjadi kegagalan kita.

## Ruang lingkup

- Klien untuk **21 Connect REST**, dengan **batas waktu tunggu** dan **percobaan ulang** yang
  disetel per integrasi.
- Pemutus arus (circuit breaker) agar layanan yang sedang mati tidak dipanggil berulang.
- Pencatatan setiap panggilan: tujuan, hasil, lama.
- Kredensial dan endpoint diambil dari konfigurasi lingkungan, **bukan dari kode**.

## Non-goal

- **Tidak** menggarap layanan masuk — itu `TKT-S4-002`.
- **Tidak** menggarap pengganti DB Link — itu `TKT-S4-003`.
- **Tidak** mengubah kontrak yang dipakai sistem lawan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **18 dari 21 Connect REST ber-`pyUseAuthentication=false`** — dibawa apa adanya atau diperbaiki? | **Work Owner + Tim Infra/Security** (`R-18`) | Memperbaikinya menuntut sistem lawan ikut berubah. Membiarkannya berarti memindahkan kerentanan ke sistem baru dengan sengaja |
| **`GetPremiumPaid_SPK` memakai `http://` ke IP:port tanpa TLS** untuk data premi | **Tim Infra/Security** (`R-18`) | Data premi melintas jaringan tanpa enkripsi. Menyalakan TLS menuntut sisi lawan menyediakannya |
| **Endpoint BRI menunjuk sandbox** — itukah yang berjalan di produksi? | **Work Owner + tim integrasi** (`R-18`) | Bila ya, integrasi itu sebenarnya **tidak pernah hidup**, dan lingkupnya berubah |
| **Berapa lama batas tunggu dan berapa kali percobaan ulang per integrasi?** | **Work Owner + tim integrasi** | Percobaan ulang pada operasi yang mengubah data dapat **menggandakan transaksi** bila lawannya tidak idempoten |

## Acceptance criteria

- [ ] Seluruh **21** integrasi keluar terpanggil dari Go, dan **jumlahnya dilaporkan sebagai angka**.
- [ ] **Sistem lawan mati tidak menjatuhkan Claim PNC** — diuji dengan lawan yang sengaja
      dimatikan: proses bisnis tetap dapat diselesaikan atau ditunda dengan pesan yang jelas.
- [ ] Batas waktu tunggu **selalu ada**; tidak ada panggilan yang menggantung tanpa batas — diuji
      dengan lawan yang sengaja lambat.
- [ ] Percobaan ulang **tidak dilakukan pada operasi yang mengubah data** kecuali lawannya
      dinyatakan idempoten — diperiksa per integrasi.
- [ ] Endpoint dan kredensial **tidak ada di dalam kode** — pemindaian otomatis: **nol temuan**.
- [ ] Setiap panggilan **tercatat**: tujuan, hasil, lama — diuji.
- [ ] Gerbang 2: UAT oleh **pemilik masing-masing integrasi**.

## Dependency / Blocked by

`TKT-F1-004`. **Terhalang tiga keputusan keamanan (`R-18`).**

## Constraint keamanan, data, operasional

- **`http://` tanpa TLS untuk data premi** adalah paparan nyata, bukan catatan teknis.
- **Membawa `pyUseAuthentication=false` apa adanya** berarti memindahkan kerentanan dengan sadar.
  Bila itu yang dipilih, ia harus **tercatat sebagai keputusan**, bukan terjadi karena tidak ada
  yang menanyakannya.
- Nilai kredensial dan alamat produksi **tidak boleh masuk dokumen yang di-commit**.
- Percobaan ulang yang salah pada pengiriman ke kasir dapat menyebabkan **pembayaran ganda**.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema klaim; menambah tabel catatan panggilan.

**Rollback:** mengembalikan versi klien. Panggilan yang sudah terkirim ke sistem lain
**tidak dapat ditarik** — rollback tidak membatalkan efeknya di sana.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/rest/... -run TestBatasWaktuTungguSelaluAda
go test ./internal/adapter/rest/... -run TestLawanMatiTidakMenjatuhkanAplikasi
go run ./cmd/tools/cek-integrasi --harap 21        # HARUS 21
go run ./cmd/tools/pindai-rahasia .                # HARUS nol temuan
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 21 berkas Connect REST | direktori `Connect REST/` (dihitung langsung) |
| 18 dari 21 ber-`pyUseAuthentication=false` | `Connect REST/*.xml` (dihitung langsung) · `R-18` |
| 9 Connect REST baru ditemukan setelah inventaris v1.0 | `docs/verifikasi-bukti-adr.md:2159` |
| `GetPremiumPaid_SPK` `http://` tanpa TLS; endpoint BRI sandbox | `docs/verifikasi-bukti-adr.md:2784` |
| Angka 12 dikoreksi menjadi 21 di Steering dan BRD | `D-73` · `docs/Steering/21-RIWAYAT-REVISI.md` §7 |

## Comments
