package sqlstore

import (
	"fmt"
	"time"

	"claim-pnc/internal/platform/waktu"
)

// NumberPrefix menandai asal sebuah nomor laporan.
//
// # Bentuknya adalah keputusan BARU, bukan peniruan
//
// Nomor laporan di sistem lama adalah `pyID` pada case
// `ASM-FW-GCNMFW-Work-ReceiveDocument`, dan bentuknya ditentukan `pyWorkIDPrefix` pada
// rule kelas — yang **tidak ada di export**. Pencarian terhadap seluruh export tidak
// menemukan satu pun contoh nilainya. Jadi bentuk lamanya tidak diketahui, dan tidak
// dapat ditiru.
//
// Yang dipakai karena itu mengikuti SATU-SATUNYA keputusan penomoran yang sudah pernah
// diambil proyek ini, `D-71` untuk nomor klaim: tiga segmen dipisahkan titik, diawali
// penanda asal, diikuti dua digit tahun dan nomor urut.
//
//	PNCN.26.0148   nomor klaim      (D-71)
//	LPK.26.0148    nomor laporan    (berkas ini)
//
// Penanda `LPK` membuat asal sebuah nomor terbaca langsung tanpa tabel pemetaan — alasan
// yang sama yang `D-22` pakai untuk memilih `PNCN`.
//
// **Belum dikonfirmasi Work Owner.** Bila bentuk lain yang dikehendaki, yang berubah
// hanya konstanta ini dan fungsi di bawahnya.
const NumberPrefix = "LPK"

// SequenceWidth adalah banyaknya digit nomor urut, dengan nol di depan.
//
// # Ini memperbaiki cacat yang `D-71` catat pada nomor klaim
//
// Sintaks nomor klaim yang ditetapkan `D-71` memakai `TO_CHAR(seq.NEXTVAL)` **tanpa
// format mask**, sehingga lebar segmen terakhirnya berubah-ubah: `.9` lalu `.10` lalu
// `.1000`. Akibatnya pengurutan sebagai teks TIDAK sesuai urutan penerbitan — `.10`
// mendahului `.9` — dan setiap layar yang mengurutkan berdasarkan nomor akan salah.
//
// Tabel modul ini baru dan nomornya belum pernah terbit, sehingga cacat itu tidak perlu
// diwarisi. Lebar tetap membuat pengurutan teks sama dengan urutan penerbitan.
//
// Empat digit menampung 9.999 laporan per tahun. Bila terlampaui, nomornya melebar
// menjadi lima digit dan pengurutan teks kembali menyimpang pada tahun itu — dicatat
// terbuka, bukan disembunyikan. Kolomnya dibuat cukup lebar untuk menampungnya
// (migrasi `0003`), sehingga yang terjadi adalah pengurutan yang melenceng, bukan
// penyisipan yang ditolak.
const SequenceWidth = 4

// BuildNumber membentuk nomor laporan dari waktu pencatatan dan nomor urut.
//
// Tahunnya diambil dari waktu PENCATATAN yang sudah dipegang aplikasi, bukan dari
// `SYSDATE` basis data. Dua sebabnya:
//
//  1. `09-DATABASE-STRATEGY.md` §4 melarang `SYSDATE` karena tidak portabel ke
//     PostgreSQL, dan `D-71` sendiri mencatat pemakaiannya sebagai pengecualian dialek
//     yang harus dipagari. Di sini pengecualian itu tidak dibutuhkan sama sekali.
//  2. Waktu pencatatan datang dari seam Jam, sehingga nomor yang terbit dapat diuji
//     secara deterministik — termasuk perilakunya di sekitar pergantian tahun.
//
// Tahunnya tahun **WIB**, bukan UTC. Lihat waktu.TwoDigitYearWIB.
func BuildNumber(recordedAt time.Time, sequence int64) string {
	return fmt.Sprintf("%s.%02d.%0*d",
		NumberPrefix,
		waktu.TwoDigitYearWIB(recordedAt),
		SequenceWidth,
		sequence,
	)
}
