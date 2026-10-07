package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// NewSampleRepo membentuk penyimpanan berisi data contoh.
//
// # Apa yang dijamin data ini
//
// Ia mencakup SELURUH jalur layar, termasuk empat yang paling mudah terlewat:
//
//   - **Satu klaim dengan DUA fasilitas kredit.** Kunci barisnya harus tetap berbeda;
//     bila kunci dirangkai dari nomor klaim saja, tabel di layar akan menampilkan salah
//     satunya dua kali dan yang satunya hilang.
//   - **Baris berlini SURETY BOND.** Ia hanya muncul ketika penyaring Business Name
//     DIBALIK, dan itulah satu-satunya cara membuktikan "SURETY BOND" meniadakan
//     Asuransi Kredit alih-alih memilih Surety Bond.
//   - **Baris tepat di batas rentang tanggal**, diregistrasi pukul 23.40. Ia hilang
//     seketika bila batas atas diperlakukan `<=` terhadap tengah malam.
//   - **Kolom bernilai kosong.** Kolom laporan regulator memang bisa kosong, dan layar
//     harus tetap terbaca ketika itu terjadi.
//
// Seluruh nilainya KARANGAN. Tidak ada nomor polis, nama tertanggung, maupun nomor
// rekening nyata di sini (`D-69`).
func NewSampleRepo() *Repo {
	return NewRepo(
		WithRows(monitoringslinkojk.SegmentD01, sampleD01()),
		WithRows(monitoringslinkojk.SegmentF06, sampleF06()),
	)
}

// sampleDate menyusun waktu registrasi contoh.
//
// UTC, sejalan dengan `08-TECHNICAL-STRATEGY.md` §4.4: seluruh waktu disimpan UTC dan
// dikonversi ke WIB hanya di satu tempat.
func sampleDate(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Kedua kolom tanggal ini SENGAJA bernilai sama. Di produksi keduanya
// memang terisi dari kolom yang sama — lihat d01Columns.
// Klaim yang SAMA dengan baris di atas, fasilitas kredit KEDUA.
// Keterangan KOSONG — kolom laporan memang bisa kosong.
// Lini SELAIN Asuransi Kredit — hanya muncul pada pilihan "SURETY BOND".
// Di LUAR rentang contoh Maret — memastikan penyaring tanggal benar-benar
// menyaring, bukan meloloskan seluruhnya.
func sampleD01() []Record { return sampledata.Must[[]Record](sampleJSON, "sampleD01") }

// sampleF06 menyusun baris contoh segmen F06.
//
// Hanya kedelapan kolom bersumber yang diisi — sama persis dengan yang dihasilkan
// penyimpanan SQL. Mengisi ketiga puluh kolom lainnya di sini akan membuat layar tampak
// lengkap saat dikembangkan lalu kosong saat dijalankan terhadap Oracle, dan perbedaan
// itu baru ketahuan di produksi.
// sampleF06 menyusun baris contoh segmen F06.
//
// Ia MENURUNKAN barisnya dari sampleD01, dan itu bukan kemalasan: grid kedua segmen
// memakai daftar kolom yang sama (lihat f06Columns), sehingga dua salinan yang berbeda
// hanya akan membuat salah satunya usang tanpa ada yang menyadarinya.
//
// Yang ditambahkan adalah kunci khusus BERKAS EKSPOR F06 — `nomor_cif_debitur`,
// `jenis_kelamin`, `tanggal_lahir`, `alamat`, `kode_pos`, `telepon` — yang tidak tampil
// di grid tetapi dipakai ekspornya.
//
// Di produksi keduanya berbeda asal: D01 membaca tabel SLIK yang SUDAH tersusun, F06
// membaca data klaim SUMBERNYA. Perbedaan itu tidak dapat ditiru repo dalam memori, dan
// memang bukan yang diujinya.
func sampleF06() []Record { return sampledata.Must[[]Record](sampleJSON, "sampleF06") }
