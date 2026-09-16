---
title: "TKT-F2-006 — Generator nomor klaim PNCN.YY.xxxx"
labels: [modul::F-2, tipe::fondasi, status::needs-info, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-006 — Generator nomor klaim `PNCN.YY.xxxx`

Status: needs-info
Kesiapan: **terhalang keputusan** — dua pertanyaan bentuk nomor belum dijawab
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-002
Requirement: FR-F2, DAT-03    Keputusan: D-22, D-71, D-76    ADR: 0005, 0009    Risiko: R-12
Rule Pega yang digantikan: pembentukan `pzInsKey` berformat `ASM-FW-GCNMFW-WORK PNC-xxxx` — kunci teknis Pega yang bocor menjadi identitas bisnis
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`); nomor yang dihasilkan diuji pengguna lewat `B-2`

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu fungsi yang menerbitkan nomor klaim sistem baru, terisolasi di satu berkas, dan **satu-satunya
tempat dengan percabangan dialek database** di seluruh aplikasi.

Nilai bisnisnya: nomor klaim muncul di surat ke tertanggung, di PLA/DLA ke koasuransi dan
reasuransi, serta di pelaporan — dan **tidak dapat diubah surut**. Prefix `PNCN` juga membuat asal
sebuah klaim terbaca langsung selama masa paralel, ketika dua sistem menerbitkan klaim bersamaan.

## Ruang lingkup

- Fungsi penerbit nomor berformat **`PNCN.YY.xxxx`** dengan sintaks Oracle yang ditetapkan `D-71`:

  ```sql
  'PNCN' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(POOLDATA.CLAIM_NO_NONPEGA_SEQ.NEXTVAL)
  ```

- Isolasi di satu berkas (`internal/adapter/sqlstore/nomor_klaim.go`) dengan padanan PostgreSQL
  yang siap dipakai: `nextval('pooldata.claim_no_nonpega_seq')` + `to_char(current_date,'YY')`.
- Pembacaan nomor **kedua format** — `PNCN.YY.xxxx` dan warisan `PNC-xxxx` — karena keduanya hidup
  berdampingan permanen.
- Permintaan pembuatan sequence `POOLDATA.CLAIM_NO_NONPEGA_SEQ` ke DBA lewat prosedur `D-63`.

## Non-goal

- **Tidak** menomori ulang klaim lama. Nomor lama sudah tercetak di surat dan sudah dikirim ke
  reasuransi (`ADR-0009`).
- **Tidak** membangun alur registrasi — itu `B-2`.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang belum diputuskan | Pemilik | Akibat pada tiket ini |
|---|---|---|
| **Apakah sequence direset setiap awal tahun?** Bila tidak, nomor urut menembus pergantian tahun (`PNCN.26.8125` → `PNCN.27.8126`) dan segmen tahun menjadi penanda, bukan penghitung per tahun | **Work Owner** | Mengubah **AC** dan permintaan DDL sequence ke DBA |
| **Apakah lebar segmen terakhir dibuat tetap** (mis. `TO_CHAR(seq.NEXTVAL,'FM0000')`)? Tanpa itu, lebar berubah-ubah dan **pengurutan sebagai teks tidak sesuai urutan penerbitan** — `.10` mendahului `.9` | **Work Owner** | Mengubah AC pengurutan dan setiap layar/laporan yang mengurutkan berdasarkan nomor klaim |

`POOLDATA.CLAIM_NO_NONPEGA_SEQ` **nol kemunculan di seluruh export** — sequence-nya memang belum
ada dan akan dibuat. Itu **bukan** penghalang; ia pekerjaan DBA yang sudah diketahui bentuknya.


> **Satu pertanyaan sudah tertutup (`D-76`).** Karena `D-75` menetapkan satu database per
> entitas, sequence-nya terpisah per portal — dan Work Owner memutuskan **tidak perlu penanda
> portal**. Nomor klaim unik **di dalam satu portal**, dan **tidak dijamin unik antar portal**.
> Konsekuensinya dicatat di `D-76`.

## Acceptance criteria

> Dua butir bertanda ⚠️ berubah tergantung jawaban di atas.

- [ ] Nomor yang diterbitkan cocok dengan pola `^PNCN\.\d{2}\.\d+$` — diuji 1.000 penerbitan
      berturut-turut.
- [ ] **Tidak ada nomor ganda** pada 1.000 penerbitan dari **dua proses bersamaan** — diuji;
      keunikan datang dari sequence, bukan dari penguncian aplikasi.
- [ ] Sakelar dialek berada di **tepat satu berkas** — diuji dengan pemindaian: `NEXTVAL` dan
      `nextval(` tidak muncul di berkas lain mana pun.
- [ ] Pembaca nomor menerima **kedua format** dan dapat menyatakan asalnya (Pega atau Go) — diuji
      dengan enam contoh nomor.
- [ ] ⚠️ Perilaku pada pergantian tahun sesuai keputusan Work Owner — diuji dengan seam Clock
      (`F-5`) yang memajukan waktu melewati 31 Desember.
- [ ] ⚠️ Pengurutan nomor klaim sesuai keputusan Work Owner — diuji dengan deret `.9`, `.10`,
      `.100`.

## Dependency / Blocked by

- Bergantung pada `TKT-F2-001`, `TKT-F2-002`.
- **Terhalang dua keputusan Work Owner** di atas.
- Membutuhkan sequence dibuat DBA lewat prosedur `D-63`.

## Constraint keamanan, data, operasional

- **Tahun diambil dari `SYSDATE`**, yaitu tanggal server basis data — bukan tanggal kejadian dan
  bukan tanggal registrasi. Klaim yang terbit di sekitar pergantian tahun mengambil tahun dari jam
  server, bertaut dengan `R-12` (pergeseran zona waktu).
- Nomor klaim **tidak boleh** dibentuk di lapisan Domain — ia menyentuh database, jadi tempatnya
  Adapter.

## Migrasi skema / rollout / rollback

Menambah **satu sequence baru**; tidak mengubah tabel mana pun, sehingga tidak memengaruhi Pega.

**Rollback:** sequence dibiarkan ada dan tidak dipakai. Nomor yang telanjur terbit **tidak dapat
ditarik** — itu sebabnya kedua pertanyaan di atas harus dijawab **sebelum** klaim pertama terbit,
bukan sesudahnya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestNomorKlaim
go test ./internal/adapter/sqlstore/... -run TestNomorKlaimDuaProsesBersamaan
grep -rIn -iE "nextval" internal/ | grep -v nomor_klaim.go     # HARUS 0 baris
go test ./internal/adapter/sqlstore/... -run TestNomorKlaimPergantianTahun
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Format `PNCN.YY.xxxx` beserta sintaksnya | `D-71` · `ADR-0009` |
| Format lama `PNCN-xxxx` disupersede | `D-22` dibatasi `D-71` |
| Sequence belum ada — nol kemunculan di export | `ADR-0009` · verifikasi langsung |
| Satu-satunya sakelar dialek | `ADR-0005` §3.1 · `docs/Steering/09-DATABASE-STRATEGY.md` §3.1 |
| Prefix Pega `ASM-FW-GCNMFW-WORK` ditinggalkan | `D-22` · `CONTEXT.md` "Istilah yang sengaja tidak dipakai lagi" |

## Comments
