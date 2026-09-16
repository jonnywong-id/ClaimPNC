# 0022 — Tentukan cara menjalankan job terjadwal di aplikasi yang hidup di dua instans

Status: Proposed
Tanggal keputusan: —    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Lead Engineer + Tim Infra (mekanisme) · Work Owner (perilaku `AutoAcceptKomite`)
Jejak bukti: `D-57` (menutup `D-17`, `R-02`), `D-27`, `D-59` | `Job Scheduler/` (5 berkas) | `Agents/TATReportAgent-Agents.xml` | T-8
Terkait: ADR-0001, ADR-0008, ADR-0023, modul `S-6`

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan apa yang terhalang — **tanpa bagian
> `Keputusan`**. Jangan dijadikan dasar implementasi.

## Konteks

Daftar job terjadwal sempat tidak diketahui sama sekali (`R-02`). Pada 2026-09-09 export bertambah
dan folder `Job Scheduler/`, `Agents/`, serta `Service REST/` muncul. `D-57` menutup persoalan
inventarisnya:

| Job | Frekuensi | Jam mulai | Activity target |
|---|---|---|---|
| `JOBForKomiteKlaimPNC` | harian | **06:00:00** | **`AutoAcceptKomite`** |
| `JobHitungDeadlineToTemporaryCLose` | **mingguan** | 23:43:00 | `JobTemporaryCloseClaimPNC` |
| `JobSendAutoLODKlaimPersonal` | harian | 20:54:00 | `Act_SendAutoLODKlaimPersonal` |
| `PNCMyReportKlaim3` | harian | 08:00:00 | `ReportAI_Act` |
| `ProcessClaimKredit` | harian | 10:00:00 | `CreateClaimCredit_Table` |

Seluruhnya `pyIsEnabled=true`, `pyApplicableTo=Cluster`, `pyNodeTypesText=BackgroundProcessing`.
Seluruh activity targetnya ada di export.

Ditambah satu agent: `Agents/TATReportAgent-Agents.xml` — `pyEnable=true`,
**`pyTriggerInterval=1800`** (30 menit), **`pyBypassActivityAuthentication=true`**, merujuk
`CreateCasePNCAgent_ActButton`, `AlertAgentKasirBlmTransferKlaim`, `TransferAllCaseNotAssigned`
(ketiganya ada).

**Inventarisnya selesai; cara menjalankannya yang belum.** Pega menyediakan
`pyApplicableTo=Cluster` — jaminan bahwa satu job berjalan sekali meski ada banyak node. Go tidak
punya padanan otomatis, sementara `D-27` menuntut **minimal dua instans** di belakang load
balancer.

Temuan **T-8** mempertajam persoalan ini: sistem lama **tidak punya satu pun Dynamic System
Setting**. Konfigurasi dinamisnya berupa tabel Oracle yang **dikunci per IP aplikasi** — pola yang
justru bertabrakan dengan dua instans.

## Opsi yang dipertimbangkan

**Opsi 1 — Penjadwal di dalam binary Go, dengan penguncian lewat database.** Setiap instans
menjalankan penjadwal; yang berhasil mengambil kunci menjalankan job.

- Satu artefak, konsisten dengan ADR-0001; tidak ada komponen baru di produksi.
- Menuntut penguncian yang benar-benar aman, termasuk saat instans mati di tengah job — kelas
  kesalahan yang sulit diuji dan sulit terlihat.

**Opsi 2 — Cron sistem operasi di satu VM**, memanggil endpoint atau subperintah aplikasi.

- Sederhana dan mudah dipahami; tidak ada risiko job ganda.
- Menjadikan satu VM sebagai titik tunggal kegagalan untuk seluruh penjadwalan, dan menempatkan
  jadwal di luar aplikasi — tidak ikut ter-commit, tidak ikut ter-review.

**Opsi 3 — Instans penjadwal terpisah**: satu proses yang sama binary-nya, dijalankan dengan
sakelar sebagai penjadwal.

- Memisahkan beban job dari pelayanan pengguna, dan tetap satu artefak.
- Menambah satu proses yang harus dipantau dan di-deploy, meski binary-nya sama.

## Yang harus diputuskan Work Owner, bukan teknis

1. **`AutoAcceptKomite` menyetujui komite otomatis setiap hari jam 06:00** — tanpa pengguna sama
   sekali. `D-59` menetapkan satuan izin adalah menu; job ini **melewati kontrol menu apa pun**.
   Apakah perilaku ini dibawa ke sistem baru?
2. **Tiga job ber-`pyDescription = "job jalan 2 menit"`** sementara konfigurasinya `Daily`/`Weekly`
   — selisih 720× sampai 5.040×. Mana yang benar: deskripsinya atau konfigurasinya?
3. **`pyBypassActivityAuthentication=true`** pada agent — diterima untuk sistem baru?

## Konsekuensi bila dibiarkan tidak diputuskan

- Tiket `S-6` tidak dapat ditulis lengkap.
- Bila penguncian salah dirancang, `AutoAcceptKomite` dapat berjalan **dua kali** — menyetujui
  komite dua kali pada klaim yang sama. Ini kesalahan bernilai uang, bukan gangguan operasional.
- Jadwal sebenarnya tidak dapat dipastikan selama pertanyaan 2 belum dijawab, sehingga uji
  kesetaraan `S-6` tidak dapat dirancang.

## Pertanyaan terbuka

- Ketiga pertanyaan Work Owner di atas (`AutoAcceptKomite`, deskripsi vs konfigurasi, bypass
  autentikasi).
- Mekanisme mana — Opsi 1, 2, atau 3? Pemilik: Lead Engineer + Infra.
- Bagaimana kegagalan job diketahui (notifikasi, catatan, dashboard)? Di sistem lama tidak ada
  bukti mekanisme apa pun untuk ini. Pemilik: Lead Engineer + Work Owner.
