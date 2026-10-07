package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/platform/money"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// NewSampleStore membentuk penyimpanan berisi baris contoh.
//
// Dipakai saat aplikasi berjalan tanpa Oracle. Seluruh isinya KARANGAN — tidak ada satu
// pun nomor polis, nama tertanggung, atau nomor klaim nyata (`D-69`).
func NewSampleStore() *Store { return NewRepo(SampleRecords()...) }

// SampleRecords adalah sepuluh klaim contoh.
//
// # Lima di antaranya sengaja TIDAK muncul
//
// Penyimpanan contoh yang seluruh barisnya lolos tidak membuktikan apa pun: penyaring yang
// rusak pun akan tampak benar. Kelima baris berikut ada justru supaya penyaringnya dapat
// dilihat bekerja, dan masing-masing tertolak oleh sebab yang BERBEDA:
//
//	STD-0006  nilai settlement terbesarnya di BAWAH ambang
//	STD-0007  totalnya di ATAS ambang, tetapi tidak ada SATU baris pun yang melampauinya
//	STD-0008  tahun registrasinya di luar rentang yang biasa dipakai saat mencoba
//	STD-0009  kode bisnisnya termasuk yang DIKELUARKAN dari cakupan NONMBU
//	STD-0010  Group Panel-nya tidak termasuk cakupan mana pun selain "semua"
//
// STD-0007 yang paling perlu dibaca. Totalnya Rp 8 miliar — di atas ambang — tetapi
// terpecah menjadi dua baris settlement Rp 4 miliar, dan `EXISTS` di kueri lama menguji
// SATU baris, bukan jumlahnya. Tanpa baris seperti ini, keliru membaca ambang sebagai
// "jumlah seluruh settlement" akan lolos dari setiap uji.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Ketiga nilai di bawah sengaja KOSONG, bukan nol.
//
// Di basis data itu terjadi ketika kolom sumbernya NULL pada seluruh baris
// settlement: `SUM` atas kolom yang seluruhnya NULL mengembalikan NULL.
// Layar menggambarnya sebagai tanda hubung, dan berkas CSV sebagai sel
// kosong — bukan sebagai "Rp 0,00", yang akan ikut terhitung dalam
// penjumlahan di pengolah angka.
// ── Lima baris yang seharusnya TIDAK muncul ─────────────────────────────────
// Tertolak AMBANG: nilai terbesarnya Rp 4,9 miliar.
// Tertolak AMBANG meski TOTALNYA Rp 8 miliar: dua baris @ Rp 4 miliar, dan
// `EXISTS` menguji SATU baris. Inilah saksi yang membedakan pembacaan ambang
// yang benar dari pembacaan "jumlah seluruh settlement".
// Tertolak TAHUN pada rentang 2024–2025.
// Tertolak pada cakupan NONMBU: kode bisnisnya ada di daftar yang DIKELUARKAN.
// Ia tetap muncul bila Bisnis dibiarkan kosong — dan itulah yang membuatnya
// berguna: perbedaan antara "semua" dan "NONMBU" menjadi terlihat.
// Group Panel di luar keempat cakupan. Muncul hanya bila Bisnis dibiarkan
// kosong — bukti bahwa "semua" benar-benar berarti tanpa penyaring.
func SampleRecords() []Record { return sampledata.Must[[]Record](sampleJSON, "SampleRecords") }

// rupiah membentuk penunjuk nilai uang dari angka rupiah utuh.
//
// Ia mengembalikan PENUNJUK karena nil dan nol berbeda artinya pada modul ini — lihat
// catatan pada CaseStudyRow.
func rupiah(amount int64) *money.Money {
	value := money.FromRupiah(amount)
	return &value
}

// percent membentuk penunjuk persentase dikali 10.000.
//
// `percent(1000000)` adalah 100%, `percent(600000)` adalah 60%, dan `percent(4000)`
// adalah 0,4%. Bentuk ini dipakai supaya empat desimalnya utuh tanpa pecahan biner
// (`D-51`).
func percent(e4 int64) *int64 { return &e4 }

// date membentuk penunjuk tanggal UTC.
//
// Tetap, tidak dibaca dari jam sistem: data contoh yang berubah setiap kali dijalankan
// membuat uji yang bergantung padanya gagal pada hari yang tidak terduga.
func date(year int, month time.Month, day int) *time.Time {
	value := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &value
}
