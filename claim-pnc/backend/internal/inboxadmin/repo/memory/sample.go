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

// SampleOwner adalah login pemilik baris contoh pada tab yang ScopedToCaller.
//
// Ia dipakai adapter memori saja. Pada basis data sungguhan, pemiliknya adalah pengguna
// yang benar-benar membuat atau ditugasi barisnya.
const SampleOwner = "ADMINKLAIM"

// day membentuk tanggal UTC tanpa jam, supaya contoh terbaca dan hitungan Aging-nya
// mudah diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleRows adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap tab memperoleh baris yang membuat penyaringnya
// dapat dibuktikan bekerja.
//
//   - Tab ALL memuat empat lini bisnis sekaligus, sehingga dropdown Business dapat diuji.
//   - Kedua tab Unregistered RCV memuat baris ber-KURIR dan tanpa KURIR, sehingga
//     pemisahannya terlihat.
//   - Tab yang ScopedToCaller memuat satu baris MILIK ORANG LAIN, sehingga penyaring
//     kepemilikan yang lupa dipasang akan langsung terlihat sebagai baris yang bocor.
//   - Tanggalnya berjarak supaya kolom Aging tidak semuanya bernilai sama.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// ---------------------------------------------------------------- tab ALL
// Bonding — satu-satunya baris yang lolos saringan BONDING, dan satu-satunya
// yang DIBUANG saringan NONMBU meski Group Panel-nya termasuk keempatnya.
// ------------------------------------------- tab Unregistered RCV dan RCV Online
// ------------------------------------------------------- tab Request Survey
// Milik ORANG LAIN. Ia tidak boleh muncul bagi SampleOwner — bila muncul,
// penyaring kepemilikan tidak terpasang.
// ------------------------------------------------------ tab Request Dokumen
// ------------------------------------------------------- tab All Case Admin
// --------------------------------------------------------- tab Branch Claim
// ------------------------------------------------------ tab Status RCL/PUCL
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
