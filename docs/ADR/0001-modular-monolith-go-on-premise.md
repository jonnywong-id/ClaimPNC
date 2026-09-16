# 0001 — Bangun satu modular monolith Go yang dijalankan di VM on-premise

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-08`, `D-09`, `D-10`, `D-27`, `D-29` | `docs/verifikasi-bukti-adr.md` §9, §10
Terkait: ADR-0002, ADR-0003, ADR-0004, ADR-0022, modul `F-1`

## Konteks

Aplikasi lama adalah Pega PRPC 8.3 — satu platform yang menyatukan alur kerja, antarmuka,
integrasi, dan penjadwalan. Penggantinya harus berjalan di tempat yang sama: **data center
Sinarmas**, bukan cloud publik.

Tiga fakta membentuk keputusan ini:

- **Tidak ada Kubernetes dan tidak boleh diasumsikan ada** (`D-08`). Target deployment adalah
  VM atau bare metal.
- **Tim adalah developer Pega internal yang dilatih ulang** (`D-09`). Prioritasnya learning curve
  landai dan pola seragam, bukan arsitektur paling canggih.
- **Beban data besar, konkurensi rendah** (`D-10`): ribuan klaim per bulan, data historis puluhan
  juta baris, tetapi hanya **200–300 pengguna aktif harian**.

Sementara itu `D-27` menuntut ketersediaan **24/7 termasuk saat deployment**, yang berarti
minimal dua instans di belakang load balancer dan aplikasi yang **stateless**.

## Opsi yang dipertimbangkan

1. **Modular monolith Go**, satu binary, modul dipisah lewat batas paket dan seam repository.
2. **Microservices** per domain (klaim, komite, dokumen, laporan).
3. **Monolith tanpa batas modul** — paling cepat ditulis, paling cepat rusak.

## Keputusan

Aplikasi dibangun sebagai **satu modular monolith Go**, dikompilasi menjadi **satu binary** yang
dijalankan di VM on-premise, **tanpa mengasumsikan orkestrator kontainer apa pun**.

Batas modul ditegakkan di dalam kode — bukan lewat jaringan — mengikuti pembagian `F-*`, `B-*`,
`S-*`, `U-*` pada Module Breakdown. Aplikasi wajib **stateless**, menyediakan **health check
endpoint** dan **graceful shutdown**, sehingga dua instans dapat di-update bergantian.

## Rationale

Konkurensi rendah dengan volume data besar adalah profil yang **tidak** menuntut pemisahan
proses. Microservices akan menambah biaya operasional (service discovery, tracing terdistribusi,
transaksi lintas layanan) tanpa menyelesaikan masalah yang benar-benar dimiliki sistem ini —
yaitu kueri berat di atas data puluhan juta baris.

Satu binary juga satu-satunya bentuk yang realistis dijalankan tim yang baru meninggalkan Pega:
satu artefak untuk di-deploy, satu proses untuk dipantau, satu tempat untuk mencari galat.

Batas modul tetap ditegakkan agar kelak, bila benar-benar dibutuhkan, satu modul dapat dipisah
tanpa membongkar seluruhnya.

## Konsekuensi

### Positif

- Deployment sederhana: salin satu binary, jalankan ulang bergantian di dua VM.
- Transaksi database lintas modul tetap dapat dibuat **atomik dalam satu proses** — prasyarat
  bagi ADR-0007.
- Tidak ada biaya belajar orkestrator bagi tim yang sedang belajar Go dan React sekaligus.

### Negatif / utang teknis

- **Seluruh modul di-deploy bersama.** Perbaikan kecil di satu laporan tetap menuntut rilis
  seluruh aplikasi.
- **Skala hanya vertikal per instans.** Satu modul yang boros memori — pembuatan Excel bervolume
  besar pada `S-2` — memengaruhi seluruh proses.
- Batas modul hanya sekuat disiplin tim. Tanpa pemeriksaan otomatis atas ketergantungan
  antarpaket, monolith modular berubah menjadi monolith biasa tanpa ada yang menyadarinya.
- Penjadwalan job tidak lagi mendapat mekanisme cluster bawaan seperti `pyApplicableTo=Cluster`
  milik Pega — konsekuensi ini ditangani ADR-0022, dan **belum terselesaikan**.

### Risiko yang diterima secara sadar

- Bila kelak satu modul benar-benar membutuhkan skala terpisah, pemisahannya adalah pekerjaan
  rekayasa tersendiri, bukan konfigurasi.
- `D-29` menetapkan RPO/RTO mengikuti kebijakan backup korporat, sementara `D-27` menuntut 24/7.
  Keduanya menjawab hal berbeda, dan **kesesuaiannya belum diverifikasi** ke tim infra.

## Pertanyaan terbuka

- Apakah kebijakan backup korporat yang berlaku hari ini memang sejalan dengan tuntutan 24/7
  (`D-29`)? Pemilik: Tim Infra. Selama belum dijawab, target pemulihan aplikasi tidak dapat
  dinyatakan di tiket mana pun.
- Bagaimana dua instans berbagi berkas sementara (hasil export besar) bila tidak ada storage
  bersama? Pemilik: Lead Engineer + Infra. Menghalangi bagian export bervolume besar di `S-2`.
