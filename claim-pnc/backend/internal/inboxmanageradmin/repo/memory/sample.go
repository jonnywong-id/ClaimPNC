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
// Tidak satu pun nomor polis, nama tertanggung, nomor klaim, maupun nama petugas di bawah
// berasal dari data nyata. `D-69` melarang menulis data nasabah ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian untuk "data contoh".
//
// Namanya sengaja dibuat jelas-jelas karangan — "Tertanggung Contoh", "PT Contoh" — supaya
// tidak ada yang mengira ia salinan produksi bila kelak terbaca di layar pengembangan.
//
// # Apa yang dibuktikan susunan ini
//
// Sembilan baris, dipilih supaya setiap penyaring modul ini punya baris yang LOLOS dan
// baris yang TERTOLAK olehnya. Uji yang seluruh barisnya lolos tidak membuktikan
// penyaringnya bekerja.
//
//	baris  membuktikan
//	-----  ----------------------------------------------------------------------------
//	1, 2   tab Non-MBU berisi penugasan pada unit organisasi AdminPNC
//	3, 4   tab PA berisi penugasan pada unit organisasi AdminPA
//	5      tab Travel berisi penugasan pada unit organisasi AdminTRAVEL
//	6      klaim SELESAI tidak muncul di tab mana pun
//	7      klaim DITOLAK tidak muncul di tab mana pun
//	8      berkas penerimaan dokumen (kelas berbeda) tidak bocor ke tab mana pun
//	9      penugasan pada unit organisasi LAIN tidak muncul di tab mana pun
//
// Baris 2 sengaja punya tanggal pendaftaran KOSONG. Ia membuktikan dua hal sekaligus:
// pengurutan tidak panik pada tanggal kosong, dan kolom Lama Waktu Klaim tampil kosong
// alih-alih berbunyi "0 minutes ago" untuk baris yang tanggalnya memang belum ada.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Waktu dasar dibuat tetap, bukan `time.Now()`. Urutan baris pada uji karena itu tidak
// berubah menurut hari, dan uji yang memeriksanya tidak gagal esok hari tanpa ada yang
// menyentuh kode.
// Sumber bisnis dan tanggal pendaftaran sengaja KOSONG: keduanya memang
// dapat kosong pada klaim yang snapshot polisnya belum lengkap, dan layar
// harus menampilkannya sebagai tanda pisah — bukan sebagai kolom yang
// gagal dimuat.
// Klaim SELESAI. Unit organisasinya Non-MBU, sehingga bila penyaring status
// kerja terlewat ia akan muncul di tab pertama — dan uji akan menangkapnya.
// Klaim DITOLAK, dengan alasan yang sama seperti baris di atas.
// Berkas penerimaan dokumen — kelas objek kerja BERBEDA, unit organisasinya
// sama. Ia membuktikan penyaring kelas benar-benar dipakai.
// Unit organisasi LAIN. Ia membuktikan ketiga tab tidak menampilkan penugasan
// unit mana pun di luar ketiganya — termasuk unit yang namanya mirip.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
