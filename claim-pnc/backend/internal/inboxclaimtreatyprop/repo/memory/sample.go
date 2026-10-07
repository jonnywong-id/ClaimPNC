package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleRows adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Seluruh isinya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor polis, nama tertanggung, atau nama Ceding Co di bawah berasal dari
// data nyata. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan
// itu berlaku penuh pada data contoh — berkas contoh justru yang paling mudah tersalin ke
// tempat lain.
//
// Yang TIDAK dikarang adalah BENTUKNYA: susunan kunci objek kerja, penanda `CLMP`, nama
// akun antrean teknik, dan pembagian worklist/workbasket seluruhnya mengikuti export.
// Itulah yang membuat penyaring di memory.go benar-benar teruji.
//
// # Apa yang sengaja diuji oleh susunan baris di bawah
//
//	tiga baris di worklist              tab Admin harus memberi ketiganya, tanpa
//	                                    memandang siapa petugasnya
//	tiga baris di workbasket            tab Teknik harus memberi ketiganya, tab lain nol
//	satu baris berkelas objek kerja     harus tersaring di SELURUH tab
//	  yang BUKAN klaim treaty
//
// Baris `CLMP-3001` sengaja berada di antrean bernama LAIN, bukan `TreatyinPNCTeknik`. Dulu
// ia harus tersaring; sekarang ia harus MUNCUL — Report Definition antrean teknik tidak
// menyaring menurut nama antrean. Ia satu-satunya baris contoh yang perannya berbalik saat
// sumber data berpindah, dan itu ditulis di sini supaya pembalikannya disengaja.
//
// Satu baris (CLMP-2002) sengaja mengosongkan "Last update" dan "Status Claim ID". Keduanya
// dibaca lewat LEFT JOIN ke tabel objek kerja, sehingga baris yang objek kerjanya tidak
// terbaca memang mengembalikan keduanya NULL — dan layar harus menggambarnya sebagai tanda
// pisah, bukan sebagai sel yang tampak rusak.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Dua baris antrean teknik. Keduanya TIDAK membawa Subjectivity: kedua kueri
// mengirimkannya NULL karena nama kolom tereksposnya belum diketahui, dan
// penyimpanan ini meniru kueri — bukan meniru Report Definition yang memilihnya.
// Baris worklist yang objek kerjanya berkelas LAIN.
// Kelasnya BUKAN klaim treaty. Ia harus tersaring di SELURUH tab — inilah
// yang membuktikan pembatas `PXOBJCLASS` benar-benar berjalan, bukan sekadar
// tertulis.
// Baris workbasket milik antrean LAIN. Ia harus MUNCUL di tab Teknik —
// membuktikan tidak ada lagi penyaring nama akun antrean, sesuai Report
// Definition-nya.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
