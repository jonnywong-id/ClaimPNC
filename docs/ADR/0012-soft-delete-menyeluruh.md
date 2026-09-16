# 0012 — Terapkan soft delete menyeluruh; tidak ada penghapusan fisik data bernilai bisnis

Status: Accepted
Tanggal keputusan: 2026-09-13    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-66` (menyupersede `D-65`), `D-28`, `D-62` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` | `RDB List/InsertClaimPNC-SQL.xml:77` | `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`
Terkait: ADR-0013, ADR-0026, ADR-0027, modul `B-2`, `S-5`, `S-8`

## Konteks

Sistem lama menghapus baris secara fisik di banyak tempat — **termasuk pada tabel lognya sendiri**:

| Operasi | Objek | `berkas:baris` |
|---|---|---|
| `UPDATE` | `pooldata.claim_service_log` | `RDB List/UpdateLogServiceClaim-SQL.xml:27` |
| `DELETE` | `POOLDATA.JSON_KLAIM_LOG` | `RDB List/InsertClaimPNC-SQL.xml:77` |
| `UPDATE` | `T_LOGINCOAS` | `RDB List/BrowseOldEmailCoas-SQL.xml:69` |
| `UPDATE` | `pooldata.mst_login_surveyor` | `RDB List/UpdateMasterLoginSurvey-SQL.xml:9` |
| `RDB-DELETE` · `OBJ-DELETE` | berbagai | 12 step di 7 activity · 2 step di 2 activity |

Log yang dapat diubah dan dihapus bukan log. Ini bertabrakan langsung dengan `D-28`, yang
menetapkan jejak audit bersifat append-only.

`D-65` semula mencatat jawaban "sama dengan sistem lama" dan mengusulkan pemisahan tiga kelas
data sebagai jalan keluar. Work Owner **merevisi jawabannya** sebelum usulan itu disetujui, dan
revisi itulah yang menjadi `D-66`.

## Opsi yang dipertimbangkan

1. **Soft delete di mana pun** — tidak ada `DELETE` fisik pada data bernilai bisnis.
2. **Hard delete untuk data tertentu** yang disebutkan Work Owner.
3. **Sama dengan sistem lama.**

## Keputusan

**Soft delete berlaku menyeluruh.** Tidak ada `DELETE` fisik pada data bernilai bisnis di sistem
baru. Penghapusan dinyatakan lewat penanda — kolom flag beserta waktu dan pelakunya — bukan lewat
pembuangan baris.

Seluruh operasi `UPDATE`/`DELETE` pada tabel log di atas **tidak dibawa** ke sistem baru.

ADR ini menyupersede `D-65`. Pemisahan tiga kelas data yang diusulkan `D-65` **tidak diperlukan
dan tidak dipakai**.

## Rationale

Dengan tidak adanya penghapusan fisik, jejak audit append-only (`D-28`) dan kebijakan penghapusan
berdiri di atas prinsip yang sama — pertentangan yang membuat `D-65` rumit hilang dengan
sendirinya.

Ini juga satu-satunya kebijakan yang konsisten dengan **T-14**: perubahan nilai uang klaim di
sistem lama **tidak punya jejak audit sama sekali**. Menambahkan jejak audit sambil tetap
mengizinkan penghapusan fisik akan menghasilkan jejak yang tetap bisa dihilangkan.

## Konsekuensi

### Positif

- Tidak ada data bernilai bisnis yang dapat hilang tanpa jejak.
- Jejak audit (ADR-0026) menjadi kontrol yang benar-benar mengikat — penting karena `D-59`
  menghapus pemisahan tugas, sehingga audit adalah satu-satunya kontrol pengimbang.
- Retensi data menjadi keputusan kebijakan (`D-62`), bukan efek samping operasi harian.

### Negatif / utang teknis

- **Setiap kueri harus menyaring baris yang ditandai terhapus.** Satu kueri yang lupa
  menyaringnya akan menampilkan data yang seharusnya hilang — kelas cacat baru yang tidak ada
  di sistem lama.
- **Tabel tumbuh permanen.** Dengan data historis puluhan juta baris (`D-10`), indeks dan rencana
  eksekusi harus dirancang menyadari adanya baris mati.
- **Pola hapus-lalu-sisip-ulang tidak dapat dipertahankan.** `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc:492-503`
  menghapus **12 tabel** milik satu klaim lalu menyisipkan ulang seluruh pohonnya sebagai
  mekanisme idempotensi. Penggantinya belum diputuskan — ADR-0013.
- Keunikan kunci alami menjadi rumit: baris yang "terhapus" masih menempati nilai kuncinya.

### Risiko yang diterima secara sadar

- **Uji kesetaraan tidak boleh membandingkan jumlah baris.** Soft delete tidak mengubah hasil yang
  dilihat pengguna, tetapi mengubah isi tabel. Perkakas `S-8` membandingkan **hasil kueri sesuai
  aturan bisnis**; perbandingan berbasis `COUNT(*)` akan selalu berbeda dan **bukan** indikasi
  cacat.
- Soft delete **tidak** ditambahkan ke daftar perbaikan sadar `D-49` (ADR-0017), karena ia
  mengubah cara penyimpanan, bukan hasil yang terlihat.
- Dua tabel pada blok konversi (`T_DLALIST`, `T_PLALIST`) **sudah berpotensi menduplikasi baris
  hari ini** — delete-nya dikomentari di `:497`, `:498` sementara insert-nya tetap aktif di
  `:1296`, `:1325`. Cacat itu sudah ada sebelum keputusan ini.

## Pertanyaan terbuka

- Siapa yang berwenang melihat data yang sudah ditandai terhapus, dan lewat layar apa? Pemilik:
  Work Owner. Tanpa ini, soft delete hanya menyembunyikan data tanpa manfaat pemulihan.
- Apakah ada titik waktu ketika baris bertanda terhapus benar-benar dibuang (arsip atau purge),
  mengikuti `D-62`? Pemilik: Work Owner + Compliance.
