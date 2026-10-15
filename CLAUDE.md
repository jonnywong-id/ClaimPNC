# claim-pnc-context — aturan kerja

Repo dokumen proyek migrasi Claim PNC (Pega PRPC 8.3 → Golang + React, Oracle 19c → PostgreSQL): Steering, Decision Log, ADR, BRD, ticketing, dan korpus export Pega. Tidak ada kode aplikasi di sini — kode ada di `../claim-pnc-backend` dan `../claim-pnc-frontend`. Pengantar umum: `docs/AGENTS.md`.

## Korpus legacy — read-only

`Activity/` `Agents/` `Binary File/` `Connect REST/` `Correspondence/` `Data Transform/` `DataPage/` `Database/` `Flow/` `Flow Action/` `Function/` `HTML/` `Harness/` `InboxAutoClaim/` `JSON/` `Job Scheduler/` `Navigation/` `Property/` `RDB List/` `Report Definition/` `Sample/` `Sample Form/` `Section/` `Service REST/` `Ticket/` `When/` adalah export rule Pega + `.prc`/`.fnc` Oracle — **sumber kebenaran perilaku**. Baca dan kutip; jangan diubah, dipindah, atau di-rename.

- Nama berkas export bisa menyesatkan (mis. `Ticket/SendToPIC-Ticket.xml` berisi rule `SendToPICTravel`) — selalu periksa nama rule di dalam XML, jangan menyimpulkan dari nama berkas.
- Folder `Ticket/` berisi **Ticket rule** Pega (ADR-0021). Kata "tiket" di dokumen proyek selalu berarti tiket pekerjaan di `docs/ticketing/`.

## Peta dokumen

| Mencari | Buka |
|---|---|
| Blueprint migrasi (per bab) | `docs/Steering/NN-*.md` — indeks bab di `docs/Steering/README.md` |
| Keputusan bernomor `D-xx` | `docs/Steering/00-DECISION-LOG.md` — grep `^### D-<nomor>` |
| Glosarium istilah domain | `docs/Steering/CONTEXT.md` |
| Alasan arsitektural | `docs/ADR/00NN-*.md` (indeks: `docs/ADR/README.md`) |
| Kebutuhan bisnis | `docs/BRD/BRD.md` |
| Tiket implementasi per modul | `docs/ticketing/` (F-* fondasi, B-* bisnis) — cara kerja di `docs/agents/issue-tracker.md` |
| Arsip versi lama | `docs/arsip/` — jangan dijadikan rujukan |

**Jangan membaca dokumen gabungan utuh.** `docs/Steering/STEERING.md` (~600 KB) dan `docs/ADR/ADR.md` adalah **hasil bangunan** `docs/tools/build-*.js` — tidak disunting langsung (perubahan hilang saat dibangun ulang). Sunting berkas sumber per bab/ADR, lalu bangun ulang sesuai `docs/tools/README.md`. Untuk mencari, grep heading di berkas sumber lalu baca bagian yang relevan saja.

| Bab sumber | Topik | Bab sumber | Topik |
|---|---|---|---|
| 02 | Pemahaman bisnis | 09 | Database |
| 03–04 | Arsitektur as-is & to-be | 10 | API |
| 05 | Domain model, master data | 11 | Security, autentikasi, otorisasi |
| 06 | Module breakdown & urutan migrasi | 12–13 | Crosscutting, deployment |
| 07 | Strategi migrasi | 14–15 | Testing, NFR |
| 08 | Strategi teknis & standar coding | 16 | Risiko (`R-xx`) |

## Menulis di repo ini

- **Entri Decision Log yang sudah ada tidak disunting.** Perubahan pikiran ditulis sebagai keputusan **baru** yang menyebut entri yang disupersede. Koreksi fakta ditandai coret (`~~…~~`) + tanggal + sumber, bukan dihapus.
- **Setiap klaim tentang sistem existing menyebut sumbernya**: rule Pega (`berkas:baris` / nama rule), tabel/prosedur Oracle, atau nomor D-/ADR-/R-.
- **Jangan mengisi kekosongan pemahaman bisnis dengan asumsi.** Yang belum diputuskan ditulis terbuka beserta pemilik keputusannya.
- **Nilai sensitif tidak pernah ditulis**: kredensial, kunci API, hostname/IP produksi, data nasabah. Alamat email selalu disamarkan; Operator ID boleh ditulis lengkap (D-69).
- Bahasa Indonesia untuk isi dokumen; istilah domain mengikuti `docs/Steering/CONTEXT.md` tanpa perkecualian. Nama berkas baru huruf kecil dengan tanda hubung, kecuali mengikuti pola penomoran folder yang sudah ada.
