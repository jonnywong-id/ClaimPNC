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
// Yang TIDAK dikarang adalah BENTUKNYA: susunan kunci objek kerja, awalan `CLMNP-`, nama
// akun antrean teknik, dan pembagian worklist/workbasket seluruhnya mengikuti export.
// Itulah yang membuat penyaring di memory.go benar-benar teruji.
//
// # Apa yang sengaja diuji oleh susunan baris di bawah
//
//	dua baris milik `ADMINNONPROP1`     tab Admin tanpa "See All" harus memberi dua
//	satu baris milik `ADMINNONPROP2`    baris itu muncul HANYA dengan "See All"
//	satu baris TANPA nomor polis        hanya muncul saat "See TBA Claim" tercentang,
//	                                    dan ia MILIK `ADMINNONPROP1` supaya kombinasi
//	                                    "TBA tanpa See All" benar-benar ada isinya —
//	                                    kombinasi yang di sistem lama tidak terlayani
//	dua baris di workbasket teknik      tab Teknik harus memberi dua, tab lain nol
//	satu baris ber-awalan `CLMP-`       harus tersaring di SELURUH tab; ia membuktikan
//	                                    penyaring awalan tidak ikut menangkap klaim
//	                                    treaty PROPORSIONAL milik layar saudaranya
//	satu baris ber-awalan `KMTNP-`      harus tersaring pula; ia objek kerja komite,
//	                                    yang disebut fragmen GetWorkCNP_Act tetapi tidak
//	                                    pernah benar-benar dibaca kueri Admin
//	satu baris workbasket bukan teknik  harus tersaring di tab Teknik
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Kedua isian master id sengaja BERBEDA pada satu baris.
//
// Ia mewakili keadaan yang tidak dapat dikesampingkan: kolom objek kerja
// dan nilai di dalam blob JSON adalah dua sumber yang berbeda, dan belum
// ada yang pernah memeriksa apakah keduanya selalu sepakat (`R-08`).
// Baris ini yang membuat layar ketahuan bila kelak seseorang menyatukan
// keduanya diam-diam.
// Baris TBA: nomor polisnya BELUM ADA.
//
// Ia milik `ADMINNONPROP1` dengan sengaja. Di sistem lama kombinasi "TBA tanpa
// See All" tidak pernah menjalankan kueri apa pun, sehingga tidak ada data contoh
// yang pernah membuktikannya; baris inilah yang membuat selisih terencana itu
// benar-benar teruji, bukan hanya dinyatakan.
// Kosong, bukan diisi tanda apa pun: di basis data ia NULL.
// Dua baris antrean teknik. Keduanya TIDAK membawa JSONMasterID, karena kueri
// Teknik memang tidak mengambilnya.
// Klaim treaty PROPORSIONAL, milik layar saudaranya. Ia harus tersaring di seluruh
// tab — inilah yang membuktikan penyaring awalan `CLMNP-` benar-benar berjalan dan
// tidak ikut menangkap `CLMP-`.
// Objek kerja KOMITE non-proporsional. `GetWorkCNP_Act` menyebut awalan ini di
// fragmen penyaringnya, tetapi ketiga kueri Admin yang benar-benar dijalankan
// hanya menyaring `CLMNP-%`. Baris ini memastikan modul mengikuti kuerinya, bukan
// fragmen yang tidak pernah terpakai.
// Baris workbasket milik antrean LAIN. Ia harus tersaring di tab Teknik —
// membuktikan penyaring nama akun antrean berjalan, bukan sekadar "ada di
// workbasket".
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
