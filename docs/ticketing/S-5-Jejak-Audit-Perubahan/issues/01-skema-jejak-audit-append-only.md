---
title: "TKT-S5-001 — Skema jejak audit append-only dan hak akses database"
labels: [modul::S-5, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-S5-001 — Skema jejak audit append-only dan hak akses database

Status: ready-for-human
Kesiapan: siap
Modul: S-5 · Gelombang: 2 · Bergantung pada: TKT-F2-001, TKT-F2-004
Requirement: FR-S5    Keputusan: D-28, D-62    ADR: 0026    Risiko: R-14
Rule Pega yang digantikan: **tidak ada padanan** — sistem lama tidak mencatat perubahan nilai. Yang paling mendekati: `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` (**hanya 4 kolom**). Anti-pola yang tidak dibawa: `RDB List/UpdateLogServiceClaim-SQL.xml:27` (`UPDATE` pada log), `RDB List/InsertClaimPNC-SQL.xml:77` (`DELETE` pada log)
Peran penguji gerbang 2: **tidak berlaku** — `D-60`; gerbang 1 diganti uji fungsional terhadap kontrak (`D-56`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Tabel jejak audit yang **secara struktural tidak dapat diubah atau dihapus** oleh aplikasi, dengan
kolom yang cukup untuk menjawab *"siapa mengubah nilai ini, dari berapa menjadi berapa, kapan"*.

Nilai bisnisnya: pertanyaan itu **tidak dapat dijawab hari ini**. Dan karena `D-59` menghapus
pemisahan tugas, ia satu-satunya kontrol yang tersisa atas kewenangan menyetujui uang.

## Ruang lingkup

- Skema tabel jejak audit: entitas, id entitas, jenis aksi, nilai sebelum, nilai sesudah, pelaku,
  waktu (UTC), dan id permintaan (`TKT-F1-003`).
- **Penegakan append-only di tingkat hak akses database**: akun aplikasi hanya diberi `INSERT`
  dan `SELECT` pada tabel audit — tanpa `UPDATE` maupun `DELETE`.
- Index yang mendukung pertanyaan nyata: per entitas+id, per pelaku, per rentang waktu.
- Catatan kapasitas dan kandidat partisi per tahun, **disiapkan tetapi belum diterapkan**.

## Non-goal

- **Tidak** menentukan peristiwa apa saja yang wajib dicatat — itu `TKT-S5-003`, menunggu
  Compliance.
- **Tidak** membangun layar penelusuran audit — belum diputuskan apakah pengguna bisnis perlu
  melihatnya (`ADR-0026` pertanyaan terbuka).
- **Tidak** menerapkan partisi di awal.

## Acceptance criteria

- [ ] `UPDATE` pada tabel audit memakai akun aplikasi **ditolak database** — diuji langsung,
      bukan diasumsikan dari kode.
- [ ] `DELETE` pada tabel audit memakai akun aplikasi **ditolak database** — diuji.
- [ ] Satu baris audit memuat kedelapan kolom wajib, dan **nilai sebelum serta sesudah tidak
      boleh keduanya kosong** — ditegakkan constraint, diuji.
- [ ] `request_id` pada baris audit **sama** dengan `request_id` di log untuk permintaan yang sama
      — diuji.
- [ ] Kueri "seluruh perubahan pada klaim X" dan "seluruh perubahan oleh pelaku Y dalam rentang
      tanggal" **memakai index** — dilampirkan keluaran `EXPLAIN PLAN` keduanya.
- [ ] Migrasi skema tabel audit backward-compatible dan punya `down.sql` yang berfungsi
      (`TKT-F2-004`).

## Dependency / Blocked by

Bergantung pada `TKT-F2-001` dan `TKT-F2-004`. Membutuhkan **DBA** untuk menetapkan hak akses
akun aplikasi — itu bagian prosedur `D-63`.

## Constraint keamanan, data, operasional

- **Append-only ditegakkan hak akses database, bukan hanya kode.** Aturan yang hanya ada di kode
  bisa dilanggar oleh kode berikutnya; aturan yang ada di hak akses tidak.
- Nilai sebelum dan sesudah dapat memuat **data nasabah**. Tabel audit karena itu tunduk pada
  pembatasan akses yang sama dengan data aslinya — termasuk pembatasan data medis (`FR-R2`).
- Tabel ini **tumbuh paling cepat** di seluruh basis data (`D-10`).

## Migrasi skema / rollout / rollback

Menambah tabel baru — **tidak menyentuh tabel mana pun yang dibaca Pega**, sehingga aman bagi
masa paralel.

**Rollback:** tabel dibiarkan ada dan tidak ditulisi. **Tabel audit tidak pernah di-`DROP` sebagai
rollback** — itu menghapus bukti, dan bertentangan dengan tujuan modul ini.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
migrate -path db/migrations -database "$DSN" up
# uji hak akses dengan akun aplikasi
psql/sqlplus -U app -c "UPDATE jejak_audit SET pelaku='x' WHERE id=1;"   # HARUS ditolak
psql/sqlplus -U app -c "DELETE FROM jejak_audit WHERE id=1;"             # HARUS ditolak
go test ./internal/adapter/sqlstore/... -run TestJejakAuditAppendOnly
EXPLAIN PLAN FOR SELECT * FROM jejak_audit WHERE entitas='klaim' AND entitas_id=:1;
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Sistem lama tidak punya jejak audit atas nilai | `T-14` · `docs/verifikasi-bukti-adr.md` §10.8 |
| History hanya 4 kolom | `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` |
| Tabel log terbukti dimutasi | `RDB List/UpdateLogServiceClaim-SQL.xml:27` · `RDB List/InsertClaimPNC-SQL.xml:77` |
| Append-only ditegakkan hak akses DB | `docs/Steering/09-DATABASE-STRATEGY.md` §8 · `ADR-0026` |
| Jejak audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` |

## Comments
