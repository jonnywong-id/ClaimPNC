---
title: "TKT-B10-002 — Transfer ke kasir dan status pembayaran"
labels: [modul::B-10, tipe::integrasi, status::needs-info, prioritas::tinggi, gelombang::4]
milestone: "Gelombang 4 — Persetujuan"
epic: "Migrasi Claim PNC"
---

# TKT-B10-002 — Transfer ke kasir dan status pembayaran

Status: needs-info
Kesiapan: **terhalang artefak dan `D-40`**
Modul: **B-10 Akseptasi** · Gelombang: 4 · Bergantung pada: TKT-B10-001, TKT-S4-001
Requirement: FR-B10    Keputusan: D-25, D-40    ADR: 0008, 0025    Risiko: R-07, R-17
Rule Pega yang digantikan: `InsertLogKasir_act` dan `TransferCashierDataASM_act` — **keduanya tidak ada di export** · `Activity/HitupdateDataRekeningToKasir-Act.xml:2151` (URL kasir sebagai konstanta) · **3 Connect REST kasir dengan header `Authorization` di-hardcode**
Peran penguji gerbang 2: **PncCollection** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Data pembayaran klaim sampai ke sistem kasir **tepat satu kali**, dan status pembayarannya kembali
terlihat di klaim.

Nilai bisnisnya paling keras di sini: **pengiriman ganda ke kasir berarti pembayaran ganda**. Dan
integrasi ini bersifat asinkron, sehingga idempotensi bukan kenyamanan melainkan syarat.

## Ruang lingkup

- Pengiriman data pembayaran ke sistem kasir lewat seam integrasi (`S-4`).
- **Idempotensi**: pengiriman ulang untuk akseptasi yang sama **tidak menghasilkan pembayaran
  kedua**.
- Pembaruan status pembayaran pada klaim setelah kasir memproses.
- Data rekening penerima dari master (`MasterRekening`).

## Non-goal

- **Tidak** membangun sistem kasir.
- **Tidak** memutuskan tujuan penyimpanan kredensial — itu `D-40`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **`InsertLogKasir_act` dan `TransferCashierDataASM_act`** — tidak ada di export | **Tim Pega** (`R-07`) | Bentuk data yang dikirim ke kasir tidak diketahui |
| **Header `Authorization` di-hardcode di 3 Connect REST kasir** | **Tim Infra/Security** (`D-40`, `R-17`) | Kredensial tidak boleh dibawa apa adanya; tujuannya belum ditetapkan |
| Kontrak API kasir yang berlaku | **tim pemilik sistem kasir** | Menentukan bentuk permintaan, kode galat, dan cara idempotensi disepakati |

## Acceptance criteria

- [ ] Pengiriman ke kasir untuk satu akseptasi **tepat satu kali**; pengiriman ulang dengan kunci
      yang sama **tidak menghasilkan pembayaran kedua** — diuji dengan tiga kali pengiriman.
- [ ] Kegagalan jaringan **masuk antrean dan dicoba ulang**, dengan jeda bertambah — diuji.
- [ ] Kegagalan yang berulang **terlihat** (tercatat dan dapat diketahui), bukan hilang diam-diam.
- [ ] Pengiriman **tidak berada di dalam transaksi database** (`TKT-F2-003`) — diuji pemindaian.
- [ ] Status pembayaran yang kembali dari kasir memperbarui klaim, dan perubahannya **tercatat di
      jejak audit**.
- [ ] **Nol kredensial di kode** — diuji pemindaian; kredensial dibaca dari konfigurasi
      (`TKT-F1-002`).
- [ ] Rekening penerima diambil dari master, bukan diketik bebas — diuji.
- [ ] Gerbang 2: UAT **PncCollection** dengan pembayaran uji yang benar-benar sampai ke kasir.

## Dependency / Blocked by

`TKT-B10-001` · `TKT-S4-001` · `TKT-F4-001` (master rekening). **Terhalang Tim Pega, Tim
Infra/Security, dan tim kasir.**

## Constraint keamanan, data, operasional

- **Pengiriman ganda berarti pembayaran ganda.** Idempotensi adalah syarat, bukan optimasi — ini
  disebut eksplisit pada tabel sistem eksternal Steering: *"Kasir — asinkron, idempoten wajib;
  tidak boleh kirim ganda"*.
- Data yang dikirim memuat **nomor rekening** — data nasabah; ia tidak boleh masuk log
  (`TKT-F1-003`).
- Header `Authorization` yang kini di-hardcode adalah bagian dari `R-17`.

## Migrasi skema / rollout / rollback

Menambah tabel antrean pengiriman dan riwayatnya. Backward-compatible.

**Rollback:** pembayaran yang telanjur terkirim **tidak dapat ditarik dari sistem kasir** oleh
aplikasi ini — pembatalannya menempuh prosedur kasir.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/kasir/... -run TestKirimIdempoten
go test ./internal/adapter/kasir/... -run TestRetryDenganJedaBertambah
grep -rIn "http\." internal/app/akseptasi/ | grep -i "tx\|transaksi"     # HARUS 0 baris
grep -rInE "Authorization.*(Bearer|Basic)\s+[A-Za-z0-9]" internal/       # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `InsertLogKasir_act`, `TransferCashierDataASM_act` tidak ada di export | `docs/verifikasi-bukti-adr.md` §15 baris `B-10` · `R-07` |
| Header `Authorization` hardcode di 3 Connect REST kasir | idem |
| Kasir: asinkron, idempoten wajib, tidak boleh kirim ganda | `docs/Steering/10-API-STRATEGY.md` §8.3 |
| URL kasir sebagai konstanta | `Activity/HitupdateDataRekeningToKasir-Act.xml:2151` |

## Comments
