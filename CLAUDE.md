# claim-pnc-backend — aturan kerja

Backend Go Claim PNC (migrasi dari Pega PRPC 8.3): modular monolith (ADR-0001), satu entrypoint `cmd/claimpnc`, modul bisnis di `internal/<modul>/`, infrastruktur bersama di `internal/platform/`. Basis data Oracle 19c dulu, PostgreSQL kemudian (ADR-0005). Steering, ADR, Decision Log (`D-xx`), dan tiket ada di `../claim-pnc-context/docs/` — Steering per bab di `docs/Steering/`, ADR di `docs/ADR/`, keputusan di `docs/Steering/00-DECISION-LOG.md`, istilah domain di `docs/Steering/CONTEXT.md`.

## Menjalankan

- Salin `.env.example` menjadi `.env` di **akar repo ini** lalu isi; jalankan dari akar repo: `go run ./cmd/claimpnc` (`.env` dicari relatif terhadap folder kerja). Variabel lingkungan proses **menang** atas isi `.env`.
- `APP_ENV` = development | test | staging | production. `PENYIMPANAN` kosong → memori (data contoh) di development/test, oracle di staging/production. `IDENTITAS_ADAPTER=fake` dan `PENYIMPANAN=memori` menolak berjalan di production.
- Mode periksa integrasi (DB + HCC/HCQ, tidak menulis apa pun): `go run ./cmd/claimpnc -periksa [-login <user>]`.
- Antarmuka disematkan dari `spa/dist/` (hasil `npm run build` di frontend). `spa/dist/.gitkeep` wajib tetap ada supaya `go build ./...` jalan di clone bersih.
- Kredensial hanya hidup di env atau `.env` lokal (ADR-0025) — tidak pernah di berkas yang di-commit, termasuk `.claude/settings.json`.

## Struktur modul

Pola satu modul (contoh: `internal/masterstatus/`):

```
<modul>.go, errors.go      domain + galat bertipe
usecase/                   aturan bisnis, batas transaksi
http/                      dto.go, handler.go, errors.go (peta galat → kode HTTP), routes.go (Mount)
repo/memory/               seam memori + sample.go (data contoh)
repo/sqlstore/             SQL Oracle: teks di *.sql (`-- name: <nama>`), dimuat via go:embed
```

Rute dipasang lewat `<modul>http.Mount(...)` di dalam grup `protected` (sesi) pada `cmd/claimpnc`. `main.go` sudah ~9.000 baris — perakitan modul baru ditaruh di berkas terpisah seperti `modules.go` / `registration.go`.

## Aturan yang mengikat

- **SQL selalu berparameter** dan teksnya di berkas `.sql`, bukan string di kode Go. Tidak pernah merangkai nilai ke teks SQL.
- **Migrasi** (`migrations/NNNN_<topik>.up.sql` + `.down.sql`): hanya objek BARU berawalan `CPNC_`; tabel yang dibaca/ditulis Pega tidak disentuh (penulis tunggal, ADR-0004). Akun aplikasi tidak punya hak DDL — migrasi dijalankan DBA dengan persetujuan Work Owner (D-63). Nomor sudah pernah dipakai beberapa topik sekaligus; sebelum membuat berkas baru, periksa nomor tertinggi di cabang sendiri **dan** cabang tujuan merge.
- **Waktu disimpan UTC**; konversi ke WIB hanya di aplikasi (Steering 08 §4.4).
- **Portal multi-entitas (ADR-0030):** rute yang menyentuh DB entitas memakai `portalhttp.ActivePortal`. Permintaan tanpa portal **ditolak, tidak pernah dialihkan** ke portal utama (R-20 — kebocoran data antar badan hukum). Modul yang hanya terikat satu `*sql.DB` dijaga `onlyPrimaryPortal` (lihat `cmd/claimpnc/portalguard.go`).
- **Tanpa Oracle, SELURUH seam modul memakai memori** — tidak boleh separuh data nyata, separuh contoh.
- **Tidak ada logika bisnis di HTTP handler** maupun di `cmd/claimpnc`. Handler memetakan request/response; galat dipetakan di `http/errors.go` menjadi `{kode, pesan}` dengan kode snake_case Indonesia yang stabil (frontend membedakan lewat `kode`, bukan teks). Rincian galat internal hanya ke log, tidak ke peramban. Galat dibungkus konteks (`fmt.Errorf("...: %w", err)`); tidak ada panic di jalur request.
- **Tidak ada nilai bisnis atau endpoint yang di-hardcode** — ambil dari master/parameter DB (mis. `POOLDATA.M_PARAMETER`, `POOLDATA.GCNM_CONNECT_REST`) atau konfigurasi (ADR-0025).
- **Otorisasi berbasis menu (ADR-0023, D-59)** belum ditegakkan per rute: rute baru cukup terlindungi sesi + portal, sampai TKT-F3-005 selesai. Jangan membuat skema peran sendiri.
- **Batas modul = batas paket Go:** tipe/tabel internal satu modul tidak di-import langsung modul lain — lewat interface terekspor, dirangkai di `cmd/claimpnc`.
- **Perilaku mengikuti legacy.** Setiap aturan yang dipindahkan dari Pega menyebut sumbernya di komentar (rule Pega, tabel/prosedur Oracle, atau nomor D-/ADR-/R-). Yang belum diputuskan ditulis terbuka, tidak diisi asumsi.
- Bahasa: komentar, pesan pengguna, dan log berbahasa Indonesia, mengikuti istilah `CONTEXT.md`.

## Gerbang sebelum menyatakan selesai

```
gofmt -l . && go vet ./... && go build ./... && go test ./...
```

`gofmt -l` harus kosong. Setiap perubahan kode disertai test untuk kode yang disentuh (unit di `usecase/`, `repo/memory/`, dan `repo/sqlstore/` dengan `go-sqlmock`). CI GitLab juga menjalankan `golangci-lint run ./...` (pemblokir: govet, staticcheck SA*, ineffassign) dan `go test -race`.

- Di Windows, `go build ./...` kadang mencetak `unlinkat ...\go-build...: The directory is not empty` — itu gagal membersihkan folder temp, bukan gagal build; yang menentukan exit code-nya.
- `golangci-lint` belum terpasang di mesin lokal; bila ingin menyamai CI, pasang versi `v2.14.0` (lihat `.gitlab-ci.yml`).
- Test probe ke Oracle nyata dilewati kecuali env-nya dipasang (mis. `PROBE_INBOXSURVEY=1`, `PROBE_PORTAL`, `DDLPROBE`).
- Beberapa test membandingkan dengan berkas form frontend dan dilewati bila berkasnya tidak terbaca.

## Perkakas diagnosis (baca-saja, bukan bagian aplikasi)

- `cmd/cekddl` — `EXPLAIN PLAN` setiap kueri Report Klaim (nama tabel/kolom sah).
- `cmd/cekkueri` — menjalankan kueri lengkap dengan penyaring + parameter (menutup celah `-periksa` yang hanya `WHERE 1 = 0`).
- `cmd/cekobjek` — membedakan ORA-00942 karena objek belum dibuat, beda skema (butuh sinonim), atau tanpa GRANT.

Setelah mengubah aturan validasi, rute, nama medan JSON, atau kunci env: **jalankan ulang backend** dan pastikan tidak ada proses lama di port yang sama sebelum menguji dari frontend — `go build` tidak menangkap ketidakcocokan dengan frontend.
