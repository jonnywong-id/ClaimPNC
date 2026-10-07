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

// SampleOwner adalah login petugas pemilik rekap contoh.
//
// Ia dipakai adapter memori saja. Pada basis data sungguhan, pemiliknya adalah petugas yang
// benar-benar tercatat sebagai PIC klaimnya.
const SampleOwner = "ADMINKLAIM"

// day membentuk tanggal UTC tanpa jam, supaya contoh terbaca dan saringan tanggalnya mudah
// diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleClaims adalah baris klaim contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja.
//
//   - Dua baris SUDAH jatuh tempo dan dua belum, sehingga region Next Follow Up dapat
//     dibedakan dari Outstanding tanpa mengubah apa pun.
//   - Satu baris tindak lanjut terakhirnya BELUM PERNAH dicatat, sehingga baris yang bocor
//     ke Next Follow Up karena perbandingan terhadap nilai kosong langsung terlihat.
//   - Satu klaim berada di DUA posisi sekaligus, sehingga penggabungan antarposisi —
//     pengganti `GET_POSISI_PROGRESS_PNC` — terlihat hasilnya.
//   - Satu baris tidak punya posisi berjalan sama sekali.
//   - Nama PIC berbeda-beda, sehingga kotak cari yang lupa menelusuri kolom PIC akan
//     terlihat sebagai pencarian yang tidak menemukan apa-apa.
//   - `ProcessDate` berjarak dan satu baris dibiarkan kosong, sehingga urutannya —
//     termasuk penempatan baris tanpa tanggal di belakang — dapat diperiksa.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Sudah lewat tenggat pada tanggal contoh — muncul di Next Follow Up.
// Tepat pada tanggal contoh — jatuh tempo HARI INI, jadi ikut muncul.
// Belum jatuh tempo — tidak muncul di Next Follow Up.
// Sengaja tanpa tanggal proses: barisnya harus jatuh di BELAKANG saat diurutkan.
// Belum pernah ada catatan tindak lanjut — tidak boleh muncul di Next Follow Up.
var SampleClaims = sampledata.Must[[]ClaimRecord](sampleJSON, "SampleClaims")

// SamplePICs adalah rekap contoh.
//
// Ketiga barisnya memakai login yang sama tetapi lini bisnis berbeda, sehingga penyaring
// lini bisnis yang lupa dipasang akan langsung terlihat sebagai tiga baris yang muncul
// bersamaan. Satu baris memakai login lain, sehingga penyaring kepemilikan pun terbukti.
var SamplePICs = sampledata.Must[[]PICRecord](sampleJSON, "SamplePICs")

// NewSampleStore membentuk pembaca berisi baris contoh.
//
// Salinan, bukan senarai aslinya: dua portal yang memakai adapter memori tidak boleh
// berbagi baris yang sama, karena perubahan pada satu akan terlihat di yang lain — persis
// kebocoran antarentitas yang `R-20` peringatkan.
func NewSampleStore() *Store {
	claims := make([]ClaimRecord, len(SampleClaims))
	copy(claims, SampleClaims)

	pics := make([]PICRecord, len(SamplePICs))
	copy(pics, SamplePICs)

	return NewStore(claims, pics)
}
