# 0026 — Catat setiap perubahan bernilai bisnis sebagai jejak audit append-only

Status: Accepted
Tanggal keputusan: 2026-09-07 (`D-28`), 2026-09-13 (`D-62`)    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-28`, `D-62`, `D-59`, T-14 | `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` | `RDB List/InsertClaimPNC-SQL.xml:77`
Terkait: ADR-0012, ADR-0023, ADR-0028, modul `S-5`

## Konteks

**Perubahan nilai uang klaim di sistem lama tidak punya jejak audit sama sekali** (T-14). Yang
paling mendekati, `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc:7`, hanya mencatat **empat
kolom** — jauh dari cukup untuk menjawab "siapa mengubah nilai ini, dari berapa menjadi berapa".

Lebih buruk lagi, dua tabel yang namanya log **terbukti dimutasi**:
`RDB List/UpdateLogServiceClaim-SQL.xml:27` melakukan `UPDATE` pada `claim_service_log`, dan
`RDB List/InsertClaimPNC-SQL.xml:77` melakukan `DELETE` pada `JSON_KLAIM_LOG`.

Kepentingan jejak audit **naik drastis** karena ADR-0023: `D-59` menghapus pemisahan tugas, dan
satuan izin adalah menu. Tidak ada kontrol teknis yang mencegah satu orang membuat, menyetujui,
dan membayarkan satu klaim. **Jejak audit menjadi satu-satunya kontrol pengimbang yang tersisa.**

## Opsi yang dipertimbangkan

1. **Jejak audit sebagai persyaratan wajib yang dirancang sejak awal**, append-only.
2. Tambahkan jejak audit setelah modul bisnis selesai.
3. Ikuti sistem lama — catat seadanya.

## Keputusan

Jejak audit adalah **persyaratan wajib yang dirancang sejak awal**, bukan tambahan.

Setiap perubahan bernilai bisnis tercatat permanen — **siapa, kapan, nilai sebelum, nilai
sesudah** — minimal untuk: nilai estimasi klaim, nilai settlement, akseptasi, keputusan komite,
penolakan (RCL), proses ulang (PUCL), perubahan status klaim, dan pembayaran.

Data audit bersifat **append-only**: **tidak boleh diubah atau dihapus oleh jalur aplikasi mana
pun**. Kedua anti-pola di atas tidak dibawa ke sistem baru.

**Retensi jejak audit mengikuti retensi data klaim yang berlaku sekarang** (`D-62`) — satu
kebijakan untuk keduanya, bukan kebijakan terpisah. Sampai angkanya diperoleh, `S-5` dibangun
dengan **retensi sebagai parameter konfigurasi** (konsisten ADR-0025), sehingga modulnya tidak
terhalang.

## Rationale

Menambahkan jejak audit belakangan berarti setiap modul bisnis harus dibongkar ulang untuk
menyisipkan pencatatan pada setiap titik perubahan — pekerjaan yang lebih besar daripada
merancangnya sejak awal, dan hampir pasti menyisakan titik yang terlewat.

Append-only bukan kehati-hatian berlebihan: sistem lama membuktikan bahwa log yang dapat diubah
**memang diubah**. Sifat append-only harus ditegakkan struktur, bukan diserahkan pada disiplin.

`D-62` menghapus satu penghalang yang tampak besar — `D-28` menggantungkan retensi pada Compliance.
Ternyata kebijakannya sudah ada; yang dibutuhkan hanya angkanya.

## Konsekuensi

### Positif

- Pertanyaan "siapa mengubah nilai ini" dapat dijawab — untuk pertama kalinya dalam sejarah
  aplikasi ini.
- Kontrol pengimbang atas ketiadaan pemisahan tugas (ADR-0023) benar-benar ada.
- Soft delete (ADR-0012) dan append-only berdiri di atas prinsip yang sama, tanpa pertentangan.

### Negatif / utang teknis

- **Volume data audit dapat melampaui data bisnisnya sendiri.** Di atas data historis puluhan juta
  baris (`D-10`), ini keputusan kapasitas, bukan sekadar keputusan kepatuhan.
- Pencatatan menambah satu operasi tulis pada setiap perubahan bernilai bisnis — memengaruhi
  seluruh jalur transaksi.
- **`S-5` adalah kemampuan baru 100% tanpa baseline Pega** (ADR-0028). Tidak ada yang dapat
  dibandingkan untuk membuktikannya benar; kelulusannya bertumpu pada **daftar peristiwa wajib
  audit** yang harus disepakati Compliance dan **belum ada**.
- Append-only menuntut penegakan di tingkat hak akses database, bukan hanya di kode aplikasi —
  dan itu menyentuh kewenangan DBA, bukan tim pengembang.

### Risiko yang diterima secara sadar

- Angka retensi diambil dari kebijakan yang sudah berjalan dan **belum masuk ke repo maupun
  dokumen proyek mana pun**. `S-5` dibangun dengan parameter, dan angkanya diisi kemudian.
- Selama daftar peristiwa wajib audit belum disepakati, cakupan "perubahan bernilai bisnis" adalah
  daftar minimum di atas — bukan daftar final.

## Pertanyaan terbuka

- **Angka retensi data klaim yang berlaku sekarang** — berapa? Pemilik: Work Owner + Compliance.
  Tidak menghalangi pembangunan `S-5`, tetapi menghalangi go-live-nya.
- **Daftar peristiwa wajib audit beserta field yang harus tercatat** — pemilik: Compliance
  (`D-56`). Ini kontrak yang menggantikan gerbang 1 bagi `S-5`; tanpanya `S-5` tidak dapat lulus
  gerbang apa pun.
- Apakah jejak audit perlu dapat dibaca pengguna bisnis lewat layar, atau cukup tersedia untuk
  audit? Pemilik: Work Owner.
