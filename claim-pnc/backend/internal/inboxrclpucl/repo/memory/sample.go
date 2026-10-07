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
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "CONTOH-…" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Sebelas baris, dipilih supaya SETIAP penyaring modul ini punya baris yang LOLOS dan baris
// yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan penyaringnya
// bekerja — yang membuktikannya adalah baris yang seharusnya tidak muncul dan memang tidak
// muncul.
//
//	baris  membuktikan
//	-----  -----------------------------------------------------------------------------
//	1, 2   tab Cetak Surat berisi klaim tanpa tanggal cetak dan ber-STATUSCASE_1 '0'
//	3      STATUSCASE_1 selain '0' TIDAK muncul di tab Cetak Surat meski suratnya belum
//	       dicetak — penyaring kedua tab itu benar-benar dipakai
//	4, 5   tab Kelengkapan Dokumen berisi klaim bersurat, belum disetujui, bukan MSIG
//	6      klaim yang SUDAH disetujui tidak muncul di tab mana pun
//	7      klaim ber-PUCLAPPROVE_1 KOSONG tidak muncul di tab Kelengkapan Dokumen —
//	       meniru `NULL <> '1'` yang bernilai UNKNOWN, bukan TRUE
//	8      tab Klaim MSIG berisi klaim berpenanda MSIG, dan klaim itu TIDAK bocor ke tab
//	       Kelengkapan Dokumen
//	9      klaim SELESAI tidak muncul di tab mana pun
//	10     klaim di antrean bersama LAIN tidak muncul di tab mana pun
//	11     klaim Personal Accident di LUAR antrean RCL/PUCL — tidak muncul di tab mana pun,
//	       tetapi IKUT di laporan harian lewat cabang kedua UNION
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
// menyentuh kode.
// sent menyusun pasangan waktu kirim dan teksnya sekaligus.
//
// Keduanya sengaja berasal dari satu sumber: yang satu dipakai menyaring dan
// mengurutkan laporan, yang lain digambar layar, dan membiarkannya menyimpang akan
// membuat baris contoh menceritakan dua hal yang berbeda.
// claimAge menyusun isi kolom "Lama Klaim".
//
// # Ia TANGGAL, bukan angka — dan baris contoh ini sempat menyatakan sebaliknya
//
// Sampai 2026-09-30 kesebelas baris di bawah mengisinya dengan bilangan ("12", "5", …),
// mengikuti judul kolomnya. Judul itu menyesatkan sejak di Pega: Work Owner menjelaskan
// isinya **tanggal kirim untuk proses PUCL**, dan kolomnya terverifikasi bertipe
// `TIMESTAMP(6)` di Oracle pada hari yang sama.
//
// Baris contoh yang bentuknya berbeda dari produksi meloloskan uji yang tidak akan
// lolos di produksi, sehingga bilangan itu diganti tanggal.
//
// Ia sengaja dibuat satu detik LEBIH AWAL daripada "Tanggal Masuk Inbox", bukan sama
// persis. Di produksi kedua kolom memang terpaut milidetik — keduanya ditulis pada
// langkah yang sama — tetapi baris contoh yang membuatnya identik akan meloloskan
// tertukarnya kedua isian tanpa ketahuan.
// ---- Tab Cetak Surat: surat BELUM dicetak, STATUSCASE_1 = '0' -------------------
// Kosong — inilah yang menempatkannya di tab Cetak Surat.
// Isian layar kerja. Ketiganya HANYA dipakai Detail, bukan oleh grid mana pun.
//
// FirstObjectName mengisi DUA isian sekaligus di layar kerja — "Nama Peserta" dan
// "UP" — karena activity penyusun lampiran memang menunjuk ekspresi yang sama
// untuk keduanya.
// SENGAJA tanpa objek dan tanpa adjustment: subkueri yang tidak mengembalikan
// baris menghasilkan isian turunan KOSONG, dan itu keadaan yang sah — bukan
// kegagalan. Uji layar kerja memakai baris ini untuk membuktikannya.
// ---- TERTOLAK tab Cetak Surat: STATUSCASE_1 bukan '0' --------------------------
//
// Suratnya belum dicetak, tetapi penanda kasusnya berbeda. Ia tidak muncul di tab mana
// pun — dan itulah yang membuktikan penyaring STATUSCASE_1 benar-benar dipakai, bukan
// sekadar ikut tertulis.
// ---- Tab Kelengkapan Dokumen: bersurat, belum disetujui, bukan MSIG ------------
// ---- TERTOLAK: sudah disetujui -------------------------------------------------
// ---- TERTOLAK: PUCLAPPROVE_1 KOSONG --------------------------------------------
//
// Baris paling penting di berkas ini. Di Oracle, `NULL <> '1'` bernilai UNKNOWN — bukan
// TRUE — sehingga baris ini TIDAK muncul di tab Kelengkapan Dokumen meski suratnya
// sudah dicetak dan ia jelas belum disetujui.
//
// Perilakunya terasa keliru, dan memang mungkin keliru. Ia tetap ditiru karena yang
// diuji adalah kesetaraan dengan Pega, bukan kebenaran aturannya — dan perbaikannya
// bukan wewenang berkas ini.
// ---- Tab Klaim MSIG ------------------------------------------------------------
//
// Di produksi kolom penandanya tampaknya tidak pernah terisi, sehingga tab ini
// kemungkinan selalu kosong. Di sini ia sengaja diisi supaya tabnya punya sesuatu untuk
// dibuktikan — dan supaya terbukti pula ia TIDAK bocor ke tab Kelengkapan Dokumen.
// ---- TERTOLAK: klaim sudah selesai ---------------------------------------------
// ---- TERTOLAK: antrean bersama LAIN --------------------------------------------
// Antrean lain: proses pengisi tidak memasukkannya ke tabel datar.
// ---- Hanya untuk LAPORAN HARIAN: klaim PA di luar antrean RCL/PUCL -------------
//
// Ia TIDAK muncul di tab mana pun — antreannya bukan RCLPUCL. Tetapi ia IKUT di laporan
// harian lewat cabang kedua `UNION`, yang mengambil seluruh klaim ber-Group Panel '002'
// pada rentang tanggal yang sama tanpa melihat antreannya sama sekali.
//
// Inilah satu-satunya baris yang membuktikan laporan dan tabel memang berbeda isinya.
// Di luar antrean RCL/PUCL: tidak ada di tabel datar, tetapi TETAP ikut laporan
// harian lewat cabang kedua UNION — laporan itu membaca tabel Pega.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
