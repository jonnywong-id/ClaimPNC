# 0007 — Naikkan logika stored procedure ke Go dan pindahkan kepemilikan transaksi ke aplikasi

Status: Accepted
Tanggal keputusan: 2026-09-07 (`D-02`), 2026-09-14 (`D-68`)    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-02`, `D-68`, `D-01` | `Database/INSERT_PLADLA.prc:69,74,79,138,143,148,179,184,189,198` | `Database/UPDATEREAS.prc` | `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18`
Terkait: ADR-0004, ADR-0005, ADR-0013, modul `B-4`, `B-5`, `B-9`, `B-10`, `B-12`

## Konteks

`D-01` dan `D-02` menetapkan **tidak ada pemanggilan stored procedure** dari aplikasi baru;
seluruh SQL ditulis di kode Go dan database menjadi penyimpanan murni.

Pembacaan sumber procedure membuktikan keputusan itu bukan sekadar soal gaya. Dari 12 procedure
yang dibaca, **10 melakukan `COMMIT` sendiri**:

| Procedure | `COMMIT` | Catatan |
|---|---|---|
| `Database/INSERT_PLADLA.prc` | **9×** (`:69`, `:74`, `:79`, `:138`, `:143`, `:148`, `:179`, `:184`, `:189`) | `ROLLBACK` satu-satunya ada di handler terluar `:198` — **terjadi setelah commit**, sehingga tidak memulihkan apa pun |
| `Database/UPDATEREAS.prc` | 4× | — |

Selama procedure itu dipanggil apa adanya, **`B-4` dan `B-9` tidak dapat dibuat atomik**:
penerbitan PLA/DLA yang menempuh sembilan `COMMIT` dapat berhenti di tengah dan meninggalkan
state setengah jalan yang tidak dapat dibatalkan.

Kontrak galatnya juga bermasalah. Galat disampaikan lewat string `ErrMsg`, dan pada enam
procedure **`ErrMsg` tidak di-set pada jalur sukses** — sehingga `NULL` berarti berhasil. Pada
`Database/ADD_NEWMASTERVIRTUALACCOUNT.prc:18`, kolom yang sama membawa **nomor virtual account
sekaligus pesan galat**.

Syarat untuk meninggalkan procedure adalah memastikan tidak ada sistem lain yang memanggilnya —
dijawab `D-68`: **Claim PNC satu-satunya pemanggil.**

## Opsi yang dipertimbangkan

1. **Naikkan seluruh logika ke Go**, procedure ditinggalkan setelah modulnya pindah.
2. **Pertahankan procedure** dan panggil dari Go — perilaku identik, tetapi non-atomik ikut
   terwarisi.
3. **Tulis ulang procedure** agar tidak melakukan `COMMIT` sendiri, lalu tetap dipanggil dari Go.

## Keputusan

Seluruh logika stored procedure **dinaikkan ke Go**. Database menjadi penyimpanan murni, dan
**kepemilikan transaksi berpindah sepenuhnya ke lapisan Go**.

Karena Claim PNC adalah satu-satunya pemanggil (`D-68`), procedure tersebut **boleh ditinggalkan**
setelah modul pemiliknya lulus gerbang — tidak perlu dipelihara demi sistem lain.

Kontrak galat berbasis string `ErrMsg` **tidak ikut dibawa**. Galat disampaikan sebagai galat
bahasa Go, bukan sebagai nilai kolom.

## Rationale

Ini satu-satunya opsi yang menyelesaikan tiga persoalan sekaligus: ketergantungan pada dialek
Oracle (ADR-0005), ketiadaan atomisitas pada `B-4`/`B-9`, dan kontrak galat yang tidak dapat
dibedakan dari data.

Menulis ulang procedure agar tidak ber-`COMMIT` (opsi 3) memindahkan pekerjaan ke PL/SQL — bahasa
yang justru sedang ditinggalkan — dan tetap menyisakan logika bisnis di dua tempat.

## Konsekuensi

### Positif

- **`B-4` Spreading Reasuransi dan `B-9` PLA/DLA dapat dibuat atomik.** Penerbitan yang di sistem
  lama menempuh sembilan `COMMIT` kini dapat dibungkus satu transaksi.
- Logika bisnis berada di satu tempat, dapat diuji dengan uji otomatis biasa.
- Menghapus ketergantungan pada PL/SQL sejalan dengan target PostgreSQL (ADR-0005).

### Negatif / utang teknis

- **Logika yang sudah teruji bertahun-tahun ditulis ulang.** Setiap procedure yang dinaikkan
  adalah kesempatan baru untuk salah, pada modul yang menghitung uang.
- **62 procedure yang sudah diterima memanggil 12 objek lain yang belum diserahkan**, terberat
  `UPDATE_LOG_KONVERSI` (**162 pemanggilan**), `GETNEWID` (42), `PKG_COUNTER_PRODUCTION` (25),
  `PROCESSQUEUEDIRECT` (10). Sumbernya tetap harus diminta untuk dibaca meski objeknya kelak
  ditinggalkan.
- Perubahan dari sembilan commit menjadi satu transaksi **mengubah perilaku saat gagal**: sistem
  lama meninggalkan sebagian data, sistem baru tidak meninggalkan apa pun. Kasus uji kesetaraan
  harus dirancang menyadari ini, atau ia akan melaporkan selisih palsu.
- Transaksi yang lebih panjang menahan kunci lebih lama. Dengan 200–300 pengguna (`D-10`) risiko
  ini kecil, tetapi tidak nol pada job massal.

### Risiko yang diterima secara sadar

- `D-68` menyatakan Claim PNC satu-satunya pemanggil **tanpa verifikasi katalog**. Bila ternyata
  ada sistem lain yang memanggil, mematikan procedure akan merusaknya. Verifikasinya cukup satu
  kueri `ALL_DEPENDENCIES`, dan **belum dijalankan**.
- Cacat yang selama ini tersembunyi di balik `COMMIT` beruntun akan tampak sebagai kegagalan
  transaksi penuh — lebih benar, tetapi lebih terlihat oleh pengguna.

## Pertanyaan terbuka

- Kapan kueri `ALL_DEPENDENCIES` dijalankan untuk membuktikan `D-68` sebelum procedure
  dinonaktifkan? Pemilik: DBA. Menghalangi langkah mematikan procedure, bukan penulisan ulangnya.
- Sumber 12 objek dependensi yang belum diserahkan — kapan tersedia? Pemilik: Tim Pega/DBA.
  Menghalangi penyelesaian tiket `B-5`, `B-9`, `B-10`, `B-12` (`R-01`).
