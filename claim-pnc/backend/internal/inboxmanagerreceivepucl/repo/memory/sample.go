package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleRows adalah baris contoh untuk pengembangan lokal dan uji.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun nomor polis, nama tertanggung, nomor case, maupun nama petugas di bawah
// berasal dari data nyata. `D-69` melarang menulis data nasabah ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian untuk "data contoh".
//
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "PT Contoh" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Delapan baris, dipilih supaya setiap penyaring modul ini punya baris yang LOLOS dan baris
// yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan penyaringnya
// bekerja.
//
//	baris  membuktikan
//	-----  ----------------------------------------------------------------------------
//	1, 2   tab Receive memuat berkas ber-Group Panel 002, berjenis klaim PA
//	3, 4   tab Receive memuat pula berkas ber-Group Panel lain, berjenis klaim NONMBU
//	5      berkas tanpa Group Panel TIDAK muncul, sama seperti di kedua grid Pega
//	6      klaim (kelas berbeda) TIDAK bocor ke tab Receive
//	7, 8   tab RCL/PUCL berisi klaim di antrean RCLPUCL, satu RCL dan satu PUCL
//	9      klaim SELESAI tidak muncul di tab RCL/PUCL
//	10     klaim di antrean bersama LAIN tidak muncul di tab RCL/PUCL
//
// Baris 1 sampai 4 membawa pula isi LAYAR KERJA penerimaan dokumennya. Baris 4 sengaja
// membawanya nyaris kosong: ia berkas tanpa pasangan di tabel cermin, dan layar kerjanya
// harus tetap terbuka alih-alih dinyatakan tidak ada.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
// menyentuh kode.
// Berkas PA yang BELUM diregistrasi menjadi klaim: nomor klaim PNC-nya kosong,
// dan itu keadaan yang sah — bukan data hilang.
// Berkas ini BELUM diregistrasi menjadi klaim — nomor klaim PNC-nya kosong —
// dan alasannya diisi di sini supaya layar kerjanya menunjukkan keadaan yang
// benar-benar dihadapi petugas.
// Group Panel 006 — Fire/Property. Ia bukan PA, sehingga muncul di tab NONMBU.
// Berkas NONMBU tanpa pasangan di tabel cermin: Nama Pengirim dan Tanggal
// Terima Dokumen kosong. Ia SENGAJA ada — `LEFT JOIN` pada kueri aslinya
// membuat baris seperti ini tetap muncul, dan penyimpanan memori harus
// menunjukkan hal yang sama.
// Layar kerjanya nyaris kosong, dan itu SENGAJA: berkas ini tidak punya pasangan
// di tabel cermin, sehingga ke-13 isian yang berasal dari sana memang NULL.
// Ia harus tetap TERBUKA — bukan dinyatakan tidak ada — karena `LEFT JOIN` pada
// kueri aslinya memang membiarkannya terbaca.
// Group Panel KOSONG. Ia tidak muncul di tab Receive mana pun, persis seperti
// `GROUPPANEL_1 <> '002'` yang tidak menangkap NULL di Oracle.
// KLAIM, bukan berkas penerimaan dokumen, tetapi berada di tabel penugasan per
// orang. Ia tidak boleh muncul di tab Receive mana pun — pembedanya semata
// `PXOBJCLASS`, dan itulah yang dibuktikan baris ini.
// Jalur RCL — `RCL_PUCL_1 = '1'`.
// Jalur PUCL — `RCL_PUCL_1 = '2'`. Suratnya BELUM dicetak, sehingga
// `LetterPrintedAt` kosong. Baris seperti ini tidak akan muncul bila ketiga
// penyaring job pengingat ikut dibawa — dan itulah alasan ketiganya tidak
// dibawa; lihat catatan di inboxmanagerreceivepucl.sql.
// Klaim SELESAI di antrean yang sama. Ia tidak boleh muncul — satu-satunya
// penyaring Report Definition RCL/PUCL adalah status kerja ini.
// Klaim di antrean bersama LAIN. Ia tidak boleh muncul — dan baris inilah yang
// membuktikan penyaring antrean benar-benar dipakai, bukan sekadar tertulis.
// Tanpanya, kueri yang lupa menyaring antrean tetap lulus seluruh uji.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
