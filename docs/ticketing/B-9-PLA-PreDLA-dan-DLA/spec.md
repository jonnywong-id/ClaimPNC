# B-9 · PLA, Pre-DLA & DLA ke Reasuransi

| | |
|---|---|
| **Nama di sistem lama** | **PLA / DLA** — inbox `InboxPLADLA`, `InboxPLA_harness`; activity `LetterOfAssignment_Act`, `LetterOfAssignment2_Act`, `SendDLAAutoSaatGeneratedDLA`; procedure `INSERT_PLADLA` |
| **Kode modul** | `B-9` |
| **Gelombang** | 5 — Nilai dan pihak luar |
| **Ukuran** | 69 activity (gabungan dengan spreading) |
| **Bergantung pada** | `B-4` Spreading · `B-5` Input Estimasi |
| **Kesiapan** | **SEBAGIAN** — **lepas dari `BRD §21.4`** (`D-55`) |

## Apa yang dikerjakan modul ini

Menerbitkan **pemberitahuan bertahap** kepada koasuransi dan reasuransi tentang kerugian yang
terjadi:

| Dokumen | Kapan |
|---|---|
| **PLA** — Preliminary Loss Advice | pemberitahuan awal, berdasarkan estimasi |
| **Pre-DLA** | pemberitahuan **sebelum akseptasi** dilakukan |
| **DLA** — Definite Loss Advice | pemberitahuan final setelah nilai ditetapkan |

## Yang dilepaskan `ADR-0007`

`Database/INSERT_PLADLA.prc` melakukan **`COMMIT` sembilan kali** (`:69`, `:74`, `:79`, `:138`,
`:143`, `:148`, `:179`, `:184`, `:189`), dengan satu-satunya `ROLLBACK` di `:198` yang terjadi
**setelah** commit — sehingga tidak memulihkan apa pun.

Setelah logikanya naik ke Go, penerbitan PLA/DLA **dapat dibuat atomik**.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | Status `VALID`/`INVALID` `GET_GROUPBUSINESS_XOL` di produksi | **DBA** |
| **Keputusan** | **Tiga perilaku berbeda untuk PLA, DLA, dan PREDLA — mana yang benar?** | **Work Owner** |
| **Keputusan** | PLA tanpa nilai sah? · PLA/DLA tanpa email reasuradur sah? | **Work Owner** |
| **Keputusan** | Kunci duplikat PLA **5 kolom** versus DLA **6 kolom** — sengaja? | **Work Owner** |
| **Keputusan** | `T_PREDLALIST` masih dipakai? | **Work Owner** |

**Sudah diputuskan:** `TTGLPLADLA` yang diterima lalu dibuang **bukan cacat** — tanggal PLA/DLA
memang tanggal sistem menerbitkan dokumen (`D-49` butir 7, **direplikasi**).

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B09-001](issues/01-penerbitan-pla-predla-dla.md) | Penerbitan PLA, Pre-DLA, dan DLA | `needs-info` |
| [TKT-B09-002](issues/02-pengiriman-ke-reasuransi-dan-koasuransi.md) | Pengiriman dokumen ke reasuransi dan koasuransi | `needs-info` |
