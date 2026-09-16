# 0019 — Pertahankan Worklist dan Workbasket; bangun ulang router sebagai aturan routing

Status: Accepted
Tanggal keputusan: 2026-09-07    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-26`, `R-04`, `D-13` | `Flow/Register_Flow.xml` | `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` | `AddTJobCounterPIC_SQL`
Terkait: CONTEXT.md#Worklist, CONTEXT.md#Workbasket, ADR-0021, ADR-0023, modul `B-6`

## Konteks

Penugasan di sistem lama memakai mekanisme Pega: tabel `PC_ASSIGN_WORKLIST` (18 rule) dan
`PC_ASSIGN_WORKBASKET` (6 rule). Keduanya lenyap bersama platformnya, sehingga sistem baru harus
punya model penugasannya sendiri.

Pembagian dua model itu **nyata dipakai**, bukan warisan kosong — terbaca dari
`Flow/Register_Flow.xml`:

| Model | Tahap |
|---|---|
| **Workbasket** (`impl=WorkBasket`, router `ToWorkbasket`) | `RCL/PUCL`, `Investigator`, `Compliance` |
| **Worklist** (`impl=WorkList`) | `Input Register`, `View Polis`, `Estimation`, `Input Estimasi`, `Choose Surveyor`, `Send To Analis`, `Send To PIC Teknik`, `RCLDokter`, `Analyst Doctor` |

Algoritma penugasannya juga sudah dapat direkonstruksi: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40`
memilih petugas dengan **`ORDER BY counter_quota ASC`** — yang paling sedikit bebannya — lalu
`AddTJobCounterPIC_SQL` menaikkan pencacahnya.

## Opsi yang dipertimbangkan

1. **Pertahankan kedua konsep** — Worklist (per orang) dan Workbasket (antrean bersama).
2. Sederhanakan menjadi satu model penugasan.
3. Rancang ulang total sesuai kebutuhan bisnis sekarang.

## Keputusan

Model penugasan mempertahankan **dua konsep**:

- **Worklist** — tugas ditugaskan ke satu orang tertentu.
- **Workbasket** — antrean bersama, diambil siapa pun yang berwenang.

Delapan router Pega dibangun ulang sebagai **aturan routing milik aplikasi**: `PNCAdminRouter`,
`PNCTeknikRouter`, `RouterRCLDokter`, `KomiteRouter`, `PNCAdminRouterRCV`, `ToCurrentOperator`,
`ToWorkList`, `ToWorkbasket`.

## Rationale

`D-13` menetapkan alur kerja tetap sama agar pengguna tidak perlu dilatih ulang. Mengubah model
penugasan mengubah cara orang bekerja setiap hari — perubahan paling terasa yang bisa dilakukan
tanpa mengubah satu pun aturan bisnis.

Pembagian dua model juga bukan kebetulan: tahap yang butuh kesinambungan penanganan memakai
Worklist, tahap yang dikerjakan sebuah tim memakai Workbasket. Pembagian itu mencerminkan cara
kerja, bukan keterbatasan platform.

## Konsekuensi

### Positif

- Cara kerja pengguna tidak berubah saat modulnya berpindah.
- Algoritma pembagian beban terbaca dari sumber, bukan ditebak: yang paling sedikit bebannya
  mendapat tugas berikutnya.
- Tabel penugasan milik aplikasi menggantikan tabel engine Pega secara bersih (ADR-0004).

### Negatif / utang teknis

- **Tiga router tidak ada di export**: `PNCAdminRouter`, `PNCTeknikRouter`, `RouterRCLDokter`
  (`R-04`). Logikanya harus digali ulang — bukan disalin.
- Pencacah beban (`counter_quota`) adalah **state yang harus dijaga konsisten**; bila naik tanpa
  turun, atau gagal naik karena galat, pembagian beban menjadi timpang secara permanen.
- Penugasan menjadi tabel milik aplikasi yang harus dikunci dengan benar saat dua orang mengambil
  tugas yang sama dari satu Workbasket — persoalan konkurensi yang di sistem lama ditangani engine.
- **Hanya ada 4 workbasket di seluruh export** (T-7), sementara penjenjangan komite justru
  menugaskan ke **operator bernama**, bukan ke workbasket. Model penugasan `B-7` karenanya tidak
  seluruhnya mengikuti pola ini.

### Risiko yang diterima secara sadar

- Membawa model Pega berarti membawa keterbatasannya, termasuk ketiadaan mekanisme "ambil kembali"
  tugas yang sudah ditugaskan ke seseorang yang kemudian tidak masuk kerja.
- Penugasan ke operator bernama (T-7) membuat ketidakhadiran seseorang dapat menghentikan klaim —
  hal yang tampaknya dijawab kolom `STS_ABS` pada master komite, dan perilakunya belum dirumuskan.

## Pertanyaan terbuka

- Logika tiga router yang hilang (`R-04`) — digali dari Tim Pega, atau ditetapkan ulang oleh Work
  Owner sebagai aturan baru? Pemilik: Work Owner. Menghalangi penyelesaian tiket `B-6`.
- Apa yang terjadi pada tugas milik seseorang yang sedang tidak hadir (`STS_ABS`)? Pemilik: Work
  Owner.
- Apakah pencacah beban direset berkala? Pemilik: Work Owner. Tanpa jawaban, perilaku jangka
  panjang pembagian beban tidak dapat ditiru.
