# Claim PNC — workspace tiga repo

Migrasi aplikasi Pega PRPC 8.3 **Claim PNC** (klaim Non-MBU) ke Go / Oracle 19c (→ PostgreSQL 17+) /
React. Claude Code dijalankan dari **folder induk** yang berisi tiga repo bersebelahan (berkas ini
diimpor oleh `CLAUDE.md` folder induk).

| Repo | Isi | Aturan kerjanya |
|---|---|---|
| `claim-pnc-context/` | Steering, Decision Log (`D-nn`), ADR, BRD, ticketing, glossary, dan export rule Pega (2.634 XML + 55 `.prc` + 8 `.fnc`, **read-only**) | `claim-pnc-context/CLAUDE.md` · `claim-pnc-context/docs/AGENTS.md` |
| `claim-pnc-backend/` | Go modular monolith, modul di `internal/<modul>/`, DDL di `migrations/` | `claim-pnc-backend/CLAUDE.md` |
| `claim-pnc-frontend/` | React + TypeScript + Vite SPA, modul di `src/modules/<modul>/` | `claim-pnc-frontend/CLAUDE.md` |

Masing-masing repo punya git, branch, dan MR sendiri. Jalankan perintah git dengan
`git -C <repo> ...`.

Versi yang berlaku adalah yang tertulis di `claim-pnc-backend/go.mod` dan
`claim-pnc-frontend/package.json`, bukan angka di Steering.

**Dilarang keras membuat branch sendiri.** Claude bekerja di branch yang sedang aktif di tiap repo.
Jangan menjalankan `git branch <nama>`, `git checkout -b`, `git switch -c`, atau membuat worktree yang
memunculkan branch baru — termasuk saat akan commit di branch default. Bila pekerjaan butuh branch
lain, berhenti dan minta developer yang membuat atau memilihkannya.

**Cek branch sebelum mengubah berkas.** Sebelum mengubah berkas di sebuah repo, jalankan
`git -C <repo> branch --show-current`. Bila hasilnya `dev` atau diawali `dev-backup-`, jangan ubah
apa pun di repo itu — tampilkan peringatan ke developer bahwa repo tersebut sedang di branch
`<nama>` dan minta mereka pindah ke branch kerja secara manual. Claude tidak memindahkan branch
sendiri; lanjutkan hanya setelah developer mengonfirmasi sudah pindah.

Kedua aturan di atas juga ditegakkan hook `claim-pnc-context/.claude/hooks/jaga-branch.sh`, yang
dipasang sebagai hook `PreToolUse` di `.claude/settings.json` **folder induk**:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit|NotebookEdit|Bash|PowerShell|EnterWorktree",
        "hooks": [
          { "type": "command", "command": "bash claim-pnc-context/.claude/hooks/jaga-branch.sh" }
        ]
      }
    ]
  }
}
```

Bila sebuah tool dibatalkan dengan pesan berawalan "BHAP", sampaikan pesannya ke developer — jangan
mencari jalan memutar.

## Alur kerja

0. **Fitur baru atau perubahan perilaku: grilling dulu.** Sebelum menulis kode, jalankan skill
   `grilling` (plugin `mattpocock-skills`) bersama developer sampai setiap keputusan terjawab.
   Perbaikan kecil — ganti teks, bug yang jelas, rapikan kode tanpa ubah perilaku — langsung
   dikerjakan.
1. **Baca tiket dan keputusan dulu.** Sebelum mengubah perilaku, cari yang mengaturnya:
   - tiket modul: `claim-pnc-context/docs/ticketing/<KODE>-<Nama>/spec.md` dan `issues/`;
   - keputusan arsitektur/bisnis: `claim-pnc-context/docs/Steering/00-DECISION-LOG.md` (**grep
     `### D-`, jangan dibaca utuh**) dan `claim-pnc-context/docs/ADR/NNNN-*.md`;
   - bab Steering terkait: `claim-pnc-context/docs/Steering/01-…21-*.md`;
   - istilah domain: `claim-pnc-context/docs/Steering/CONTEXT.md`.
2. **Parity dulu (`P-5`).** Hasil yang benar adalah hasil yang sama dengan Pega. Deviasi yang sah
   hanya **13 butir perbaikan eksplisit `P-5`** (`D-49`); selebihnya wajib punya keputusan bernomor.
   Bukti diambil dari rule XML/`.prc` dengan `berkas:baris`. Yang tidak terbukti ditulis
   "**BELUM DIVERIFIKASI**" beserta pemiliknya, bukan ditebak.
3. **Ubah backend dan/atau frontend** sesuai kebutuhan. Perubahan yang menyentuh keduanya = dua MR,
   satu keputusan bernomor yang dirujuk keduanya.
4. **Catat keputusan baru** sebagai entri **baru** `D-nn` (lanjutkan nomor terakhir). Entri lama
   tidak pernah disunting; perubahan pikiran ditulis sebagai entri baru yang menyebut yang
   disupersede.

## Aturan tetap

- **Export rule Pega di `claim-pnc-context/` read-only** — tidak diubah, dipindah, direname, atau
  dihapus.
- **Nilai sensitif tidak pernah ditulis** di berkas yang di-commit (`D-69`): kredensial, kunci API,
  hostname/IP produksi, data nasabah dirujuk dengan `berkas:baris` + nama elemen. Email selalu
  disamarkan; Operator ID boleh ditulis.
- **Penamaan** (`D-80`, `D-81`): identifier, berkas, dan folder di dalam kode bahasa Inggris; nama
  folder modul memakai nama modul bisnis dari developer (backend `internal/masterrekening`, frontend
  `src/modules/master-rekening`). Komentar, field JSON API, tabel/kolom basis data, teks layar, dan
  env var/flag CLI tetap bahasa Indonesia.
- **SQL portabel** (`D-20`): parameter binding, tanpa `NVL`/`ROWNUM`/`SYSDATE`/`SELECT *`, tanpa
  pemanggilan stored procedure (`D-02`).
- **Perubahan skema** lewat DBA atas permintaan tertulis + persetujuan Work Owner (`D-63`); migrasi
  selalu backward-compatible dan punya `down` yang berfungsi.

## Rujukan lintas repo — baca saat pemicunya terjadi

- **Memberi nama** berkas, identifier, rute, env var: peta penamaan Indonesia→Inggris + `D-80`/`D-81`.
- **Hasil "tidak jalan" tanpa error, atau galat Oracle aneh**: catatan keputusan implementasi dan
  catatan pengembangan — jebakan yang sudah terbukti tercatat di sana (mis. penanda bind `go-ora`
  diikat menurut urutan kemunculan, ORA-01008).
- **Format frontend**: Prettier tanpa berkas konfigurasi — wajib
  `--no-semi --single-quote --print-width 100`.
- **Uji ke `localhost`** lewat proxy korporat: `curl --noproxy '*'`; port 8080 sering sudah terpakai.

## Pengingat: refresh URL foto/dokumen

Saat mengerjakan fitur yang menampilkan atau mengunggah foto/dokumen, **periksa dulu apakah fitur itu
sudah menangani URL kedaluwarsa** (URL diminta ulang ke layanan storage saat dibuka, URL baru
diterbitkan bila sudah lewat `EXPDATE`). Contoh yang sudah ada: modul Dokumen Penunjang —
`claim-pnc-backend/internal/dokumenpenunjang/` (`Document.Kedaluwarsa`, perpanjangan lewat
`POST /api/v1/geturl` di `storage/httpstorage`) dan
`claim-pnc-frontend/src/modules/dokumen-penunjang/DokumenPenunjangPanel.tsx`. **Sampaikan hasil
pemeriksaannya ke developer** — sudah ada atau belum — dan bila belum, tanyakan apakah perlu
ditambahkan sebelum melanjutkan.

## Pengingat: teks pesan error & validasi

Teks pesan yang tampil ke user (error, validasi, konfirmasi) **berasal dari developer atau dari Pega,
bukan dikarang Claude**. Bila pesannya ada di export Pega, pakai teksnya verbatim (`D-13`). Bila
pesannya baru, **berhenti dan minta developer menuliskan teksnya** — sebutkan kondisi pemicunya,
layar/endpoint-nya, dan kode error-nya — lalu pakai teks dari developer apa adanya.

## Jalur pada dokumen lama

Dokumen yang ditulis selagi Claim PNC masih satu repo memakai jalur lama. Bacalah:

| Jalur lama | Jalur sekarang |
|---|---|
| `docs/…` | `claim-pnc-context/docs/…` |
| `Activity/`, `RDB List/`, `Section/`, `Database/`, … (root) | `claim-pnc-context/…` |
| `claim-pnc/backend/…` | `claim-pnc-backend/…` |
| `claim-pnc/frontend/…` | `claim-pnc-frontend/…` |
| "repository terpisah `D:\…\XML Claim PNC\docs`" | `claim-pnc-context/docs/` |
| `…/Migrasi-Pega-Ke-Golang-ClaimPNC/ClaimPNC/…` (mesin developer lain) | root repo lama di atas |

Jalur berkas di catatan implementasi sebelum 2026-09-18 masih memakai nama Indonesia (sebelum
`D-80`/`D-81`); catatan itu rekaman dan tidak disunting.
