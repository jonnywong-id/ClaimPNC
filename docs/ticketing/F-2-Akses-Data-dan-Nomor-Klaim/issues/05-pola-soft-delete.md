---
title: "TKT-F2-005 — Soft delete sebagai pola akses data"
labels: [modul::F-2, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F2-005 — Soft delete sebagai pola akses data

Status: ready-for-human
Kesiapan: siap
Modul: F-2 · Gelombang: 1 · Bergantung pada: TKT-F2-001, TKT-F2-002
Requirement: FR-F2    Keputusan: D-66 (menyupersede D-65), D-28    ADR: 0012, 0026    Risiko: —
Rule Pega yang digantikan: penghapusan fisik yang tersebar — `RDB List/InsertClaimPNC-SQL.xml:77` (`DELETE` pada `POOLDATA.JSON_KLAIM_LOG`) · `RDB List/UpdateLogServiceClaim-SQL.xml:27` (`UPDATE` pada `pooldata.claim_service_log`) · `RDB List/BrowseOldEmailCoas-SQL.xml:69` · `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` · 12 step `RDB-DELETE` di 7 activity · 2 step `OBJ-DELETE` di 2 activity
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu pola penghapusan yang dipakai seluruh aplikasi: **data bernilai bisnis tidak pernah dibuang**,
penghapusan dinyatakan lewat penanda beserta waktu dan pelakunya.

Nilai bisnisnya bertaut langsung dengan `ADR-0023`: karena satuan izin adalah menu dan **tidak ada
pemisahan tugas**, jejak audit menjadi **satu-satunya kontrol pengimbang**. Jejak audit yang
barisnya masih bisa dihapus bukan kontrol apa pun — dan sistem lama membuktikan itu terjadi: dua
tabel yang namanya log terbukti dimutasi.

## Ruang lingkup

- Kolom penanda baku untuk seluruh tabel bernilai bisnis: penanda terhapus, waktu, dan pelaku.
- **Pembantu kueri** yang menyaring baris terhapus secara baku, sehingga menyaringnya menjadi
  perilaku bawaan dan **menampilkannya** yang menuntut niat eksplisit.
- **Pemeriksaan otomatis**: pernyataan `DELETE` pada tabel bernilai bisnis menggagalkan build.
- Pola penanganan **keunikan kunci alami** ketika baris terhapus masih menempati nilai kuncinya.
- Catatan indexing: index pada tabel bernilai bisnis memperhitungkan adanya baris mati.

## Non-goal

- **Tidak** menetapkan siapa yang boleh melihat data terhapus dan lewat layar apa — itu keputusan
  Work Owner yang masih terbuka (`ADR-0012`).
- **Tidak** membangun arsip atau purge — itu bertaut retensi (`D-62`), dan angkanya belum ada.
- **Tidak** menyelesaikan pola pengganti hapus-lalu-sisip-ulang — itu `ADR-0013`, masih
  `Proposed`, dan menjadi tiket `B-2`.

## Acceptance criteria

- [ ] Pembantu kueri **menyaring baris terhapus secara baku** — kueri yang ditulis tanpa
      menyebutkan apa pun **tidak** mengembalikan baris terhapus. Diuji.
- [ ] Menampilkan baris terhapus menuntut pemanggilan eksplisit — diuji bahwa jalur biasa tidak
      bisa melakukannya secara tidak sengaja.
- [ ] Pemeriksaan pola **menggagalkan build** pada `DELETE FROM` terhadap tabel bernilai bisnis —
      diuji dengan commit percobaan.
- [ ] Menghapus lalu menyisipkan ulang baris dengan kunci alami yang sama **berhasil** dan tidak
      melanggar constraint unik — diuji pada tabel contoh.
- [ ] Penanda terhapus menyimpan **waktu (UTC) dan pelaku**; keduanya wajib terisi — diuji bahwa
      penghapusan tanpa pelaku ditolak.
- [ ] Satu baris jejak audit tercatat pada setiap penghapusan (`S-5`) — diuji setelah `S-5` ada;
      sampai itu, tiket ini menyediakan titik pemanggilannya.

## Dependency / Blocked by

Bergantung pada `TKT-F2-001` dan `TKT-F2-002`.

**Yang bergantung padanya:** seluruh modul bisnis, dan `S-8` — karena perkakas uji kesetaraan
harus tahu bahwa membandingkan jumlah baris tabel akan selalu berbeda.

## Constraint keamanan, data, operasional

- Tabel **tumbuh permanen**. Di atas data historis puluhan juta baris (`D-10`), ini keputusan
  kapasitas, bukan sekadar kepatuhan.
- **Setiap kueri pembaca wajib menyaring** — satu kueri yang lupa akan menampilkan data yang
  seharusnya hilang. Ini **kelas cacat baru** yang tidak ada di sistem lama, dan itulah alasan
  penyaringan dijadikan perilaku bawaan, bukan tanggung jawab penulis kueri.
- Soft delete **tidak berlaku** pada berkas di storage eksternal (`ADR-0010`): menghapus metadata
  tidak menghapus berkasnya, dan itu keputusan tersendiri yang belum diambil.

## Migrasi skema / rollout / rollback

Menambah kolom penanda ke tabel bernilai bisnis — **tambah kolom yang *nullable***, sehingga
backward-compatible dan aman bagi Pega yang masih membaca tabel yang sama (`P-4`).

**Rollback:** kolom penanda dibiarkan ada dan diabaikan; tidak ada data yang hilang. Menghapus
kolomnya menempuh prosedur dua tahap `TKT-F2-004`.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/sqlstore/... -run TestSoftDelete
grep -rIn -E "DELETE\s+FROM" internal/ | grep -v "_test.go"   # HARUS 0 baris pada tabel bisnis
go test ./internal/adapter/sqlstore/... -run TestKunciAlamiSetelahSoftDelete
make lint-sql                                                  # pemeriksaan pola DELETE
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Soft delete menyeluruh | `D-66` · `ADR-0012` |
| `DELETE` pada tabel log | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` pada tabel log | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| 12 step `RDB-DELETE` + 2 step `OBJ-DELETE` | `docs/verifikasi-bukti-adr.md` (D-66) |
| Jejak audit satu-satunya kontrol pengimbang | `D-59` · `ADR-0023` · `ADR-0026` |
| Uji kesetaraan tidak boleh membandingkan `COUNT(*)` | `docs/Steering/14-TESTING-STRATEGY.md` §6.4 |

## Comments
