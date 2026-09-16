---
title: "TKT-U2-005 — Pemilihan pustaka tabel: TanStack Table versus AG Grid"
labels: [modul::U-2, tipe::keputusan-teknis, status::ready-for-human, prioritas::tinggi, gelombang::2]
milestone: "Gelombang 2 — Kerangka UI"
epic: "Migrasi Claim PNC"
---

# TKT-U2-005 — Pemilihan pustaka tabel: TanStack Table versus AG Grid

Status: ready-for-human
Kesiapan: siap
Modul: U-2 · Gelombang: 2 · Bergantung pada: TKT-U1-001
Requirement: FR-U2    Keputusan: D-23, D-09    ADR: 0002    Risiko: R-11
Rule Pega yang digantikan: grid bawaan Pega pada **268 dari 269 section**; `Section/` seluruhnya
Peran penguji gerbang 2: **tidak berlaku** — modul fondasi UI (`D-60`)

## Hasil yang diharapkan (dan nilai bisnisnya)

Satu keputusan tertulis tentang pustaka tabel mana yang dipakai, **beserta alasannya dan bukti
pengukurannya** — bukan preferensi.

Nilai bisnisnya: pilihan ini mengikat 268 layar. Menggantinya setelah 50 layar ditulis adalah
pekerjaan berbulan-bulan. `ADR-0002` sengaja meninggalkannya terbuka karena koreksi ukuran
(median 6 kolom, bukan 18–27) **melemahkan alasan memilih pustaka kelas berat** — dan keputusan
itu pantas diambil dengan angka, bukan dengan kesan.

## Ruang lingkup

- Prototipe kecil untuk **kedua** pustaka, memuat kasus terberat yang benar-benar ada:
  grid **39 kolom**, grid **25 kolom**, dan grid dengan 10.000 baris hasil paginasi server-side.
- Pengukuran yang dibandingkan: waktu render awal, waktu ganti halaman, ukuran bundel yang
  ditambahkan, dan **jumlah baris kode** yang dibutuhkan untuk mencapai perilaku yang sama.
- Pemeriksaan **lisensi**: AG Grid punya edisi komersial; bila fitur yang dibutuhkan ada di edisi
  berbayar, biayanya diangkat sebagai keputusan Work Owner.
- Keluaran akhir: **pembaruan `ADR-0002`** dengan keputusan dan angkanya, bukan berkas ADR baru.

## Non-goal

- **Tidak** membangun komponen tabel produksi — itu `TKT-U2-001`, dan menunggu tiket ini.
- **Tidak** membandingkan seluruh pustaka tabel yang ada. `ADR-0002` sudah mempersempit ke dua.

## Acceptance criteria

- [ ] Kedua prototipe dapat dijalankan dari repository dan **menampilkan data yang sama**.
- [ ] Tabel pengukuran memuat keempat metrik untuk kedua pustaka, dengan **angka**, bukan kesan.
- [ ] Pengukuran dijalankan pada kasus **39 kolom**, **25 kolom**, dan **10.000 baris** —
      ketiganya diambil dari layar nyata, disebutkan nama harness-nya.
- [ ] Kebutuhan lisensi dinyatakan tegas: fitur apa yang menuntut edisi berbayar, atau **nihil**.
- [ ] `ADR-0002` diperbarui: pilihan final, alasannya, dan angka pendukungnya; bagian
      **Negatif / utang teknis** ikut diperbarui dengan konsekuensi pilihan itu.
- [ ] Keputusan mempertimbangkan `D-09` secara eksplisit — pustaka yang menuntut lebih sedikit
      konsep baru bagi tim eks-Pega diberi bobot, dan bobot itu ditulis, bukan disiratkan.

## Dependency / Blocked by

Bergantung pada `TKT-U1-001` (kerangka SPA agar prototipe dapat dijalankan).

**Yang bergantung padanya:** `TKT-U2-001`, dan lewat itu seluruh layar `U-3`, `U-4`, `U-5`.

## Constraint keamanan, data, operasional

- Prototipe **tidak boleh** memakai data produksi. Data contoh dibuat sendiri — meski staging
  memuat data nyata (`ADR-0029`), prototipe tidak membutuhkannya.
- Bila pilihan jatuh pada edisi berbayar, **keputusan biaya milik Work Owner**, bukan tim teknis.

## Migrasi skema / rollout / rollback

Tidak menyentuh data maupun skema. **Rollback:** membuang prototipe; tidak ada dampak.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
cd prototipe/tanstack && npm ci && npm run build && npm run bench
cd prototipe/aggrid   && npm ci && npm run build && npm run bench
# bandingkan ukuran bundel
du -sh prototipe/*/dist
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| 268 dari 269 section bergrid | `docs/Steering/06-MODULE-BREAKDOWN.md` §4 |
| Median 6 kolom; tiga fitur grid termahal tidak dipakai | `T-11` · `docs/Steering/01-FRONTEND-ANALYSIS.md` banner koreksi |
| Pilihan TanStack versus AG Grid masih terbuka | `ADR-0002` pertanyaan terbuka |
| Tim eks-Pega, utamakan sedikit konsep baru | `D-09` |
| Grid 39 kolom dan 25 kolom | `docs/verifikasi-bukti-adr.md` §15 baris `U-2` |

## Comments
