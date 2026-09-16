---
title: "TKT-B12-002 — Recovery dan Virtual Account"
labels: [modul::B-12, tipe::integrasi, status::needs-info, prioritas::sedang, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-B12-002 — Recovery dan Virtual Account

Status: needs-info
Kesiapan: **terhalang artefak dan `D-40`**
Modul: **B-12 Salvage** · Gelombang: 5 · Bergantung pada: TKT-B12-001, TKT-S4-001
Requirement: FR-B12    Keputusan: D-02, D-68, D-40    ADR: 0007, 0008, 0025    Risiko: R-17, R-18
Rule Pega yang digantikan: `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` — **`ErrMsg` membawa nomor virtual account sekaligus pesan galat** · harness `MasterRecovery` · `Activity/Insert_salvageToGAByService-Act.xml` · Service REST `RecivedDataandAttachmentLelangASMSimasbid`
Peran penguji gerbang 2: **PncCollection** dan **PncManagerAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Dana yang kembali dari pihak ketiga — hasil lelang, pemulihan dari pihak yang bertanggung jawab —
tercatat dan tertaut ke klaimnya, sebagian lewat **Virtual Account**.

Nilai bisnisnya: recovery **mengurangi kerugian bersih** perusahaan. Dana yang masuk tanpa tertaut
klaim menjadi selisih yang harus dicari manual saat rekonsiliasi.

## Ruang lingkup

- Pencatatan recovery per klaim, termasuk sumbernya.
- Penerbitan dan penautan **Virtual Account** untuk penerimaan dana.
- Penerimaan data dan lampiran hasil lelang dari sistem luar (Service REST
  `RecivedDataandAttachmentLelangASMSimasbid`).
- **Kontrak galat yang bersih**: nomor Virtual Account dan pesan galat **tidak lagi berbagi satu
  kolom**.

## Non-goal

- **Tidak** membangun sistem lelang.
- **Tidak** memanggil `ADD_NEWMASTERVIRTUALACCOUNT` (`ADR-0007`).

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Kontrak layanan Virtual Account** dan kredensialnya | **Tim Infra/Security** (`D-40`) + tim pemilik sistem | Penerbitan VA menyentuh rekening; kredensialnya belum punya tempat |
| **4 Service REST masuk — masuk lingkup?** Salah satunya `RecivedDataandAttachmentLelangASMSimasbid` | **Work Owner** (`ADR-0008`) | Permukaan masuk ini belum pernah masuk hitungan `FR-S4` |
| Otentikasi layanan masuk: **sembilan Connect REST ber-`pyUseAuthentication=false`** | **Work Owner + tim integrasi** | Menentukan siapa boleh memanggil layanan penerimaan lelang |

## Acceptance criteria

- [ ] Recovery dapat dicatat dan **tertaut ke klaim**; recovery tanpa klaim **ditolak** — diuji.
- [ ] Penerbitan Virtual Account mengembalikan **nomor VA** dan **galat** sebagai dua hal terpisah
      — diuji; **kontrak lama yang menggabungkan keduanya tidak boleh punya padanan**.
- [ ] Penerbitan VA yang gagal **tidak meninggalkan recovery setengah jadi** — diuji.
- [ ] Layanan penerimaan data lelang **memeriksa kewenangan pemanggil** — diuji dengan permintaan
      tanpa kredensial: **ditolak**.
- [ ] Penerimaan data lelang bersifat **idempoten**: kiriman ganda tidak menghasilkan recovery
      kedua — diuji.
- [ ] **Nol pemanggilan stored procedure** dan **nol kredensial di kode** — diuji pemindaian.
- [ ] Gerbang 2: UAT **PncCollection** dengan satu penerimaan dana uji.

## Dependency / Blocked by

`TKT-B12-001` · `TKT-S4-001` · `TKT-F1-002` (konfigurasi rahasia). **Terhalang Tim Infra/Security
dan Work Owner.**

## Constraint keamanan, data, operasional

- Virtual Account menyentuh **rekening dan dana**; kredensialnya tunduk `D-40` yang masih `OPEN`.
- **Layanan masuk adalah permukaan serang.** Sembilan Connect REST ber-`pyUseAuthentication=false`
  dan satu memakai `http://` ke IP:port **tanpa TLS** untuk data premi (`R-18`) — pola itu tidak
  boleh dibawa.
- Nomor rekening dan VA adalah data nasabah; keduanya **tidak boleh masuk log**.

## Migrasi skema / rollout / rollback

Menambah tabel recovery dan Virtual Account. Backward-compatible.

**Rollback:** VA yang telanjur terbit **tetap ada di sistem bank** — pembatalannya menempuh
prosedur bank, bukan aplikasi.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/app/recovery/... -run TestRecoveryWajibTertautKlaim
go test ./internal/adapter/va/... -run TestNomorVADanGalatTerpisah
go test ./internal/adapter/http/... -run TestLayananMasukMemeriksaKewenangan
grep -rIn "CALL \|ADD_NEWMASTERVIRTUALACCOUNT" internal/       # HARUS 0 baris
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| `ErrMsg` membawa nomor VA sekaligus pesan galat | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18` · `ADR-0007` |
| 4 Service REST masuk, termasuk penerimaan lelang | `D-57` · `ADR-0008` §8.5 |
| 9 Connect REST tanpa autentikasi; satu `http://` tanpa TLS | `docs/verifikasi-bukti-adr.md` §15 baris `S-4` · `R-18` |
| Recovery lewat Virtual Account | `CONTEXT.md` — **Recovery** |

## Comments
