# Claim PNC — Migrasi dari Pega ke Golang + React

Repository ini berisi export rule Pega PRPC 8.3 aplikasi **Claim PNC** — **2.634 berkas XML**
ditambah **55 `.prc`** dan **8 `.fnc`** — beserta dokumen migrasinya ke **Golang + React** dengan
database **Oracle 19c → PostgreSQL 17+**.

**Status: Steering, BRD, dan ADR selesai; ticketing sedang disusun. Belum ada kode implementasi.**

## Peta dokumen

| Folder | Isi | Mulai dari |
|---|---|---|
| `docs/Steering/` | Blueprint migrasi — 16 bab sumber + `CONTEXT.md` + `00-DECISION-LOG.md` | [`docs/Steering/README.md`](Steering/README.md) |
| `docs/ADR/` | **29 Architecture Decision Record** — 24 `Accepted`, 5 `Proposed` | [`docs/ADR/README.md`](ADR/README.md) |
| `docs/BRD/` | Business Requirement Document | [`docs/BRD/BRD.md`](BRD/BRD.md) |
| `docs/ticketing/` | Tiket pekerjaan | *(sedang disusun)* |
| `docs/tools/` | Generator dokumen gabungan dan `.docx` | [`docs/tools/README.md`](tools/README.md) |

**Dua dokumen adalah hasil bangunan, bukan sumber:** `docs/Steering/STEERING.md` dan
`docs/ADR/ADR.md`. Menyuntingnya langsung sia-sia — perubahan hilang pada pembangunan berikutnya.
Yang disunting adalah berkas sumbernya.

## Aturan yang mengikat siapa pun yang bekerja di repo ini

1. **Rule XML Pega bersifat baca-saja.** Jangan mengubah, memindahkan, merename, atau menghapus
   satu pun berkas di `Activity/`, `RDB List/`, `Section/`, `Harness/`, `Data Transform/`, `When/`,
   `Flow/`, `Flow Action/`, `Connect REST/`, `Report Definition/`, `DataPage/`, `Function/`,
   `HTML/`, `Navigation/`, `Ticket/`, `InboxAutoClaim/`, `Database/`, `Job Scheduler/`, `Agents/`,
   `Service REST/`.
2. **Jangan menyunting entri Decision Log yang sudah ada.** Perubahan pikiran ditulis sebagai
   keputusan **baru** yang menyebut entri yang disupersede.
3. **Nilai sensitif tidak pernah ditulis** di dokumen yang di-commit: kredensial, kunci API,
   hostname/IP produksi, dan data nasabah dirujuk dengan `berkas:baris` + nama elemen saja.
   Alamat email **selalu disamarkan**; nama Operator ID boleh ditulis lengkap (`D-69`).
4. **Bahasa dokumen: Indonesia.** Istilah domain mengikuti
   [`docs/Steering/CONTEXT.md`](Steering/CONTEXT.md) tanpa perkecualian.
5. **Jangan mengisi kekosongan pemahaman bisnis dengan asumsi.** Yang belum diputuskan ditulis
   terbuka beserta pemilik keputusannya.

> **Peringatan istilah.** Folder `Ticket/` pada export berisi **Ticket rule** — mekanisme lompatan
> lateral Pega (lihat `ADR-0021`). Kata **"tiket"** di seluruh dokumen proyek ini selalu berarti
> **tiket pekerjaan** di `docs/ticketing/`.

## Agent skills

### Issue tracker

Tiket pekerjaan disimpan sebagai berkas markdown di **`docs/ticketing/`** (`D-31`) — ter-commit
dan terlihat di GitLab, **bukan** di `.scratch/`.
Lihat [`docs/agents/issue-tracker.md`](agents/issue-tracker.md).

### Triage labels

Memakai lima label kanonik apa adanya, tanpa penyesuaian.
Lihat [`docs/agents/triage-labels.md`](agents/triage-labels.md).

### Domain docs

Single-context. Glossary ada di `docs/Steering/CONTEXT.md` — **bukan** di root repo.
Lihat [`docs/agents/domain.md`](agents/domain.md).
