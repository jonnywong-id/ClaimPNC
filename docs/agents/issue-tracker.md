# Issue tracker: Local Markdown

Tiket pekerjaan dan spec untuk repo ini disimpan sebagai berkas markdown di **`docs/ticketing/`**.

> **Lokasi ini ditetapkan `D-31`, menggantikan `.scratch/`.** Alasannya: tiket akan ditinjau
> manajemen, harus ter-commit, dan diimpor ke GitLab — sedangkan nama `.scratch/` berkesan
> buangan. Bila sebuah skill menyebut `.scratch/`, baca sebagai `docs/ticketing/`.

> **Peringatan istilah.** Folder `Ticket/` di root repo berisi **Ticket rule** — mekanisme lompatan
> lateral Pega, bukan tiket pekerjaan. Jangan menulis apa pun ke sana; ia baca-saja.

## Konvensi

- Satu modul per direktori: `docs/ticketing/<kode-modul>/` — kode modul memakai penomoran yang
  sudah ada (`F-1`…`F-5`, `B-1`…`B-14`, `S-1`…`S-8`, `U-1`…`U-6`). **Jangan membuat kode baru.**
- Spec modul ada di `docs/ticketing/<kode-modul>/spec.md`
- Tiket implementasi satu berkas per tiket di
  `docs/ticketing/<kode-modul>/issues/<NN>-<slug>.md`, bernomor dari `01`, **tidak pernah** satu
  berkas gabungan
- Status triage dicatat sebagai baris `Status:` dekat bagian atas setiap berkas tiket
  (lihat `triage-labels.md` untuk daftar labelnya)
- Komentar dan riwayat percakapan ditambahkan di bagian bawah berkas di bawah heading `## Comments`
- Setiap tiket **wajib merujuk ADR yang mengikatnya** (`docs/ADR/NNNN-*.md`) dan `D-nn` yang
  menjadi dasarnya

## Saat sebuah skill berkata "publish to the issue tracker"

Buat berkas baru di bawah `docs/ticketing/<kode-modul>/` (buat direktorinya bila belum ada).

## Saat sebuah skill berkata "fetch the relevant ticket"

Baca berkas pada path yang dirujuk. Biasanya user memberikan path atau nomor tiketnya langsung.

## Operasi wayfinding

Dipakai `/wayfinder`. **Map** adalah satu berkas dengan satu berkas **anak** per tiket.

- **Map**: `docs/ticketing/<effort>/map.md` (badan Notes / Decisions-so-far / Fog).
- **Tiket anak**: `docs/ticketing/<effort>/issues/NN-<slug>.md`, bernomor dari `01`, dengan
  pertanyaannya di badan berkas. Baris `Type:` mencatat jenis tiket
  (`research`/`prototype`/`grilling`/`task`); baris `Status:` mencatat `claimed`/`resolved`.
- **Blocking**: baris `Blocked by: NN, NN` dekat bagian atas. Sebuah tiket tidak lagi terhalang
  bila setiap berkas yang disebutnya sudah `resolved`.
- **Frontier**: pindai `docs/ticketing/<effort>/issues/` untuk berkas yang terbuka, tidak
  terhalang, dan belum diklaim; nomor terkecil menang.
- **Claim**: set `Status: claimed` dan simpan sebelum mengerjakan apa pun.
- **Resolve**: tambahkan jawabannya di bawah heading `## Answer`, set `Status: resolved`, lalu
  tambahkan penunjuk konteks (ringkasan + tautan) ke Decisions-so-far pada `map.md`.
