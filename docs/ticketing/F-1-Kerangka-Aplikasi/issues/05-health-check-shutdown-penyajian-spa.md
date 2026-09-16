---
title: "TKT-F1-005 — Health check, graceful shutdown, dan penyajian SPA"
labels: [modul::F-1, tipe::fondasi, status::ready-for-human, prioritas::tinggi, gelombang::1]
milestone: "Gelombang 1 — Fondasi"
epic: "Migrasi Claim PNC"
---

# TKT-F1-005 — Health check, graceful shutdown, dan penyajian SPA

Status: ready-for-human
Kesiapan: siap
Modul: F-1 · Gelombang: 1 · Bergantung pada: TKT-F1-001, TKT-F1-002
Requirement: FR-F1    Keputusan: D-27, D-08, D-23    ADR: 0001, 0002    Risiko: —
Rule Pega yang digantikan: — Pega menyediakan ini di tingkat platform; tidak ada rule aplikasi
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Aplikasi yang dapat **di-update tanpa memutus pengguna yang sedang bekerja**, dan yang dapat
diketahui sehat-tidaknya oleh load balancer tanpa menebak.

Nilai bisnisnya diikat `D-27`: aplikasi harus tersedia **24/7 termasuk saat deployment**. Tanpa
health check dan graceful shutdown, setiap rilis memutus permintaan yang sedang berjalan — dan
pada jalur klaim, permintaan yang terputus di tengah berarti pekerjaan petugas hilang.

## Ruang lingkup

- Endpoint **liveness** (proses hidup) dan **readiness** (siap menerima lalu lintas) yang
  **terpisah** — readiness memeriksa koneksi database, liveness tidak.
- **Graceful shutdown**: pada sinyal berhenti, aplikasi berhenti menerima permintaan baru,
  menyelesaikan yang sedang berjalan sampai batas waktu, lalu keluar.
- **Penyajian SPA** sebagai berkas statis yang tersemat di binary, dengan *fallback* rute SPA:
  permintaan yang bukan `/api/...` dan bukan berkas nyata dikembalikan ke `index.html`.
- Header keamanan dasar pada penyajian statis.

## Non-goal

- **Tidak** membangun SPA-nya — itu `U-1`.
- **Tidak** menyiapkan load balancer atau konfigurasi VM — itu urusan infrastruktur.
- **Tidak** membangun metrik atau dashboard.

## Acceptance criteria

- [ ] `GET /healthz` (liveness) mengembalikan `200` **walau database mati** — diuji dengan
      database dimatikan.
- [ ] `GET /readyz` (readiness) mengembalikan **`503` saat database mati** dan `200` saat sehat —
      diuji dengan kedua keadaan.
- [ ] Saat menerima `SIGTERM`, aplikasi **menyelesaikan permintaan yang sedang berjalan** lalu
      keluar dengan kode `0`. Diuji: permintaan lambat (≥ 2 detik) yang sedang berjalan **tetap
      menerima respons lengkap**, bukan koneksi terputus.
- [ ] Batas waktu graceful shutdown dapat dikonfigurasi, dan **habisnya batas waktu** membuat
      aplikasi tetap keluar — tidak menggantung.
- [ ] `GET /` mengembalikan `index.html`; `GET /klaim/123` (rute SPA yang tidak ada di disk) juga
      mengembalikan `index.html`; `GET /api/tidak-ada` mengembalikan **galat JSON**, bukan HTML.
- [ ] Binary berjalan **tanpa berkas pendamping apa pun** — dibuktikan dengan menjalankannya dari
      direktori kosong.
- [ ] Dua instans dapat berjalan bersamaan terhadap konfigurasi yang sama tanpa saling
      mengganggu — diuji dengan menjalankan dua proses pada port berbeda.

## Dependency / Blocked by

Bergantung pada `TKT-F1-001` dan `TKT-F1-002`.

**Catatan:** SPA yang disajikan pada tahap ini boleh berupa halaman kosong berisi penanda versi.
Isi sebenarnya datang dari `U-1`.

## Constraint keamanan, data, operasional

- Endpoint health **tidak boleh** membocorkan detail internal — tanpa versi pustaka, tanpa nama
  host database, tanpa jumlah koneksi.
- Aplikasi wajib **stateless** (`D-27`): sesi tidak boleh hidup di memori satu instans, karena
  dua instans dilayani load balancer bergantian.
- **Tidak ada runtime Node.js di produksi** (`ADR-0002`) — SPA disajikan binary Go.

## Migrasi skema / rollout / rollback

Tidak menyentuh skema. **Rollback:** menjalankan binary versi sebelumnya; karena rolling
deployment, versi lama dan baru sempat hidup bersamaan — itulah alasan `P-4` mewajibkan migrasi
skema selalu backward-compatible.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/entrypoint/...
# uji shutdown: jalankan, kirim permintaan lambat, kirim SIGTERM
go run ./cmd/app & PID=$!; curl -s localhost:8080/api/lambat & sleep 1; kill -TERM $PID; wait
# uji readiness saat database mati
docker stop oracle-dev 2>/dev/null; curl -so /dev/null -w "%{http_code}" localhost:8080/readyz  # harus 503
# uji binary mandiri
mkdir -p /tmp/kosong && cd /tmp/kosong && /path/app --version
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 24/7 termasuk saat deployment; dua instans; stateless; health check; graceful shutdown | `D-27` · `docs/Steering/13-DEPLOYMENT.md` §2 |
| SPA disajikan binary Go, tanpa runtime Node.js | `ADR-0002` · `D-23` |
| Satu binary, VM on-premise | `ADR-0001` · `D-08` |
| Migrasi skema backward-compatible karena rolling deployment | `P-4` · `docs/Steering/07-MIGRATION-STRATEGY.md` |

## Comments
