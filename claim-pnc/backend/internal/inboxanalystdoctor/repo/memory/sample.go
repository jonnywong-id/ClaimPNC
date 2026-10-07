package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleOperator adalah login petugas pemilik antrean contoh.
//
// # Kenapa BUKAN `DRRATNA`
//
// Karena `DRRATNA` adalah Operator ID sungguhan yang tertanam di
// `Flow/Register_Flow.xml` (`Assignment13`, `pyOperator`), dan ia salah satu dari 24
// hardcode yang `D-15` perintahkan dihapus. Menuliskannya sebagai data contoh akan
// memindahkan hardcode itu ke sistem baru lewat pintu belakang — tepat hal yang sedang
// dihapus.
//
// `D-69` memang mengizinkan nama Operator ID ditulis di dokumen supaya tiket dapat menunjuk
// hardcode mana yang dibuang. Izin itu untuk DOKUMEN, bukan untuk data yang dijalankan.
const SampleOperator = "ADMINKLAIM"

// SampleOtherOperator adalah petugas LAIN, dipakai membuktikan batas kewenangan bekerja.
//
// Tanpa baris milik orang lain di dalam contoh, penyaring `pxAssignedOperatorID` yang hilang
// tidak akan terlihat sama sekali: seluruh baris tetap muncul, dan semuanya kebetulan benar.
const SampleOtherOperator = "ADMINLAIN"

// at membentuk waktu UTC, supaya contoh terbaca dan urutannya mudah diperiksa dengan mata.
func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleTasks adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya. Pada layar ini aturan itu lebih mengikat daripada di modul lain:
// antreannya berisi klaim Personal Accident, dan `FR-R2` memperlakukan data medis secara
// khusus.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": setiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja, dan setiap baris seperti itu diberi keterangan di tempatnya.
//
//   - Satu baris ber-`TransferFlag "1"` — milik antrean COMPLIANCE. Bila penyaring penanda
//     antrean hilang, baris itu bocor ke layar ini.
//   - Satu baris milik operator LAIN. Bila batas kewenangan hilang, ia ikut tampil.
//   - Satu baris ber-`Resolved-Completed`. Bila penyaring status terbalik menjadi `=`,
//     hanya baris itu yang tersisa.
//   - Satu baris ber-`Resolved-Rejected` yang HARUS TETAP MUNCUL — Report Definition hanya
//     mengecualikan `Resolved-Completed`, dan menyamakan keduanya adalah kesalahan yang
//     paling mudah dibuat di modul ini.
//   - Dua baris berwaktu daftar SAMA PERSIS, sehingga pemutus seri `pzInsKey` menurun dapat
//     diperiksa; tanpanya urutan keduanya tidak tetap.
//   - Satu baris dengan komentar PIC Teknis terisi, sehingga kolom yang di Oracle masih
//     kosong tetap terlihat bentuknya saat dikembangkan.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Berwaktu daftar SAMA PERSIS dengan baris di bawahnya. Keduanya ada supaya pemutus
// seri `pzInsKey` menurun dapat diperiksa — yang ber-`…0290` harus mendahului
// yang ber-`…0289`.
// HARUS TETAP MUNCUL. Report Definition mengecualikan `Resolved-Completed` saja,
// sehingga klaim yang DITOLAK tetap berada di antrean ini.
// TIDAK BOLEH MUNCUL — tugasnya sudah tuntas.
// TIDAK BOLEH MUNCUL — milik antrean Compliance, bukan antrean ini.
// TIDAK BOLEH MUNCUL bagi SampleOperator — antrean ini milik satu orang.
var SampleTasks = sampledata.Must[[]Record](sampleJSON, "SampleTasks")

// NewSampleStore membentuk pembaca berisi antrean contoh.
//
// Salinan dibuat supaya dua portal yang sama-sama memakai penyimpanan memori tidak berbagi
// senarai yang sama. Modul ini memang tidak menulis, sehingga hari ini tidak ada yang dapat
// saling menimpa — tetapi berbagi senarai antarportal adalah bentuk kebocoran yang persis
// dilarang `R-20`, dan mencegahnya sejak awal jauh lebih murah daripada menemukannya kelak.
func NewSampleStore() *Store {
	records := make([]Record, len(SampleTasks))
	copy(records, SampleTasks)
	return NewStore(records)
}
